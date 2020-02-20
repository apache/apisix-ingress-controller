package pkg

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"io"
	"encoding/json"
	"io/ioutil"
	"github.com/iresty/ingress-controller/log"
)

var logger = log.GetLogger()

func Route() *httprouter.Router{
	router := httprouter.New()
	router.GET("/healthz", Healthz)
	router.GET("/apisix/healthz", Healthz)
	//router.GET("/apisix/sync/upstream/:name", syncPodWithUpstream)
	return router
}

func Healthz(w http.ResponseWriter, req *http.Request, _ httprouter.Params){
	io.WriteString(w, "ok")
}

type CheckResponse struct{
	Ok bool `json:"ok"`
}

type WriteResponse struct{
	Status string `json:"status"`
	Msg string `json:"msg"`
}

func populateMode(w http.ResponseWriter, r *http.Request, params httprouter.Params, model interface{}) error{
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		return err
	}
	if err := r.Body.Close(); err != nil {
		return err
	}
	if err := json.Unmarshal(body, model); err != nil {
		return err
	}
	return nil
}