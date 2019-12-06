package election

import (
	"fmt"
	"time"
	"os"
	"github.com/iresty/ingress-controller/conf"
	"github.com/coreos/etcd/clientv3/concurrency"
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/iresty/ingress-controller/log"
	"context"
	"github.com/pkg/errors"
	"os/signal"
	"github.com/iresty/ingress-controller/pkg"
)

var (
	electionName     = "/ingress/election"
	candidateName    = conf.HOSTNAME
	resumeLeader     = true
	TTL              = 2
	reconnectBackOff = time.Second * 2
	session          *concurrency.Session
	election         *concurrency.Election
	client           *etcd.Client
	logger = log.GetLogger()
	started = false
)

func Elect(){
	var err error
	if candidateName == "" {
		candidateName = fmt.Sprintf("host-test-%d", time.Now().Second())
	}

	client, err = etcd.New(etcd.Config{
		Endpoints: conf.EtcdConfig.Addresses,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	// 选举结果 leaderChan是一个bool型的列表
	leaderChan, err := runElection(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	// 监听sigint中断
	//signal.Notify(c, syscall.SIGINT)
	signal.Notify(c)
	go func() {
		for {
			select {
			case <-c:
				fmt.Printf("Resign election and exit\n")
				cancel() // 中断后调用cancel，快速交还leadership
			}
		}
	}()

	for {
		select {
		case leader, ok := <-leaderChan:
			if !ok {
				return
			}
			fmt.Printf("Leader: %t\n", leader)
			conf.IsLeader = leader

			// 主服务只启动一次，不管是不是leader
			if !started {
				go pkg.Watch()
			}
			started = true
		}
	}
	cancel()
}

func runElection(ctx context.Context) (<-chan bool, error) {
	var observe <-chan etcd.GetResponse
	var node *etcd.GetResponse
	// 所有错误的chan
	var errChan chan error
	// 标识自己是不是leader
	var isLeader bool
	var err error

	// leader状态发生变化时，需要写入这个chan
	var leaderChan chan bool
	//修改leader状态
	setLeader := func(set bool) {
		// Only report changes in leadership
		if isLeader == set {
			return
		}
		isLeader = set
		leaderChan <- set
	}

	// 创建session，指定一个id，使用透传的ctx
	if err = newSession(ctx, 0); err != nil {
		return nil, errors.Wrap(err, "while creating initial session")
	}
	// 异步获取锁
	go func() {
		// 这里还有一个leaderChan
		leaderChan = make(chan bool, 10)
		// 最后关闭这个chan
		defer close(leaderChan)
		// 无线循环，用来处理选举流程 分为3个部分：默认流程、observe、reconnect
		for {
			// Discover who if any, is leader of this election
			// 查看当前的leader
			if node, err = election.Leader(ctx); err != nil {
				// ErrElectionNoLeader表示没有谁是leader，存在该错误表示第一次竞争leader
				if err != concurrency.ErrElectionNoLeader {// 除了第一次竞争leader之外，所有的错误都会导致重新竞争leader: reconnect
					logger.Errorf("while determining election leader: %s", err)
					goto reconnect
				}
			} else {
				// 如果获取到了当前的leader node
				// 判断leader是不是自己
				// 如果是自己，可以选择恢复自己的leader角色；或者重新发起一次竞争；
				if string(node.Kvs[0].Value) == candidateName { // 如果leader是自己
					// If we want to resume leadership
					if resumeLeader {
						// Recreate our session with the old lease id
						if err = newSession(ctx, node.Kvs[0].Lease); err != nil {
							logger.Errorf("while re-establishing session with lease: %s", err)
							goto reconnect
						}
						election = concurrency.ResumeElection(session, electionName,
							string(node.Kvs[0].Key), node.Kvs[0].CreateRevision)

						// Because Campaign() only returns if the election entry doesn't exist
						// we must skip the campaign call and go directly to observe when resuming
						goto observe
					} else {
						// If resign takes longer than our TTL
						// then lease is expired and we are no longer leader anyway.
						ctx, cancel := context.WithTimeout(ctx, time.Duration(TTL)*time.Second)
						election := concurrency.ResumeElection(session, electionName,
							string(node.Kvs[0].Key), node.Kvs[0].CreateRevision)
						err = election.Resign(ctx)
						cancel()
						if err != nil {
							logger.Errorf("while resigning leadership after reconnect: %s", err)
							goto reconnect
						}
					}
				}
			}
			// Reset leadership if we had it previously
			setLeader(false)

			// Attempt to become leader
			errChan = make(chan error)
			go func() {
				// Make this a non blocking call so we can check for session close
				errChan <- election.Campaign(ctx, candidateName)
			}()

			select {
			case err = <-errChan:
				if err != nil {
					if errors.Cause(err) == context.Canceled {
						return
					}
					// NOTE: Campaign currently does not return an error if session expires
					logger.Errorf("while campaigning for leader: %s", err)
					session.Close()
					goto reconnect
				}
			case <-ctx.Done():
				session.Close()
				return
			case <-session.Done():
				goto reconnect
			}

		observe:
			// If Campaign() returned without error, we are leader
			setLeader(true)

			// Observe changes to leadership
			observe = election.Observe(ctx)
			for {
				select {
				case resp, ok := <-observe:
					if !ok {
						// NOTE: Observe will not close if the session expires, we must
						// watch for session.Done()
						session.Close()
						goto reconnect
					}
					if string(resp.Kvs[0].Value) == candidateName {
						setLeader(true)
					} else {
						// We are not leader
						setLeader(false)
						break
					}
				case <-ctx.Done():
					if isLeader {
						// If resign takes longer than our TTL then lease is expired and we are no
						// longer leader anyway.
						ctx, cancel := context.WithTimeout(context.Background(), time.Duration(TTL)*time.Second)
						if err = election.Resign(ctx); err != nil {
							logger.Errorf("while resigning leadership during shutdown: %s", err)
						}
						cancel()
					}
					session.Close()
					return
				case <-session.Done():
					goto reconnect
				}
			}

		reconnect:
			setLeader(false)

			for {
				if err = newSession(ctx, 0); err != nil {
					if errors.Cause(err) == context.Canceled {
						return
					}
					logger.Errorf("while creating new session: %s", err)
					tick := time.NewTicker(reconnectBackOff)
					select {
					case <-ctx.Done():
						tick.Stop()
						return
					case <-tick.C:
						tick.Stop()
					}
					continue
				}
				break
			}
		}
	}()

	// 循环直到获取一个leader
	for {
		resp, err := election.Leader(ctx)
		if err != nil {
			if err != concurrency.ErrElectionNoLeader {
				return nil, err
			}
			time.Sleep(time.Millisecond * 300)
			continue
		}
		if string(resp.Kvs[0].Value) != candidateName {
			leaderChan <- false
		}
		break
	}
	return leaderChan, nil
}

func newSession(ctx context.Context, id int64) error {
	var err error
	session, err = concurrency.NewSession(client, concurrency.WithTTL(TTL),
		concurrency.WithContext(ctx), concurrency.WithLease(etcd.LeaseID(id)))
	if err != nil {
		return err
	}
	election = concurrency.NewElection(session, electionName)
	return nil
}