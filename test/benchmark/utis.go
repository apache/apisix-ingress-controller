// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/olekukonko/tablewriter"
	"go.uber.org/zap"
)

type TestResult struct {
	Scenario         string        `json:"scenario"`
	CaseName         string        `json:"case_name"`
	CostTime         time.Duration `json:"cost_time"`
	IsRequestGateway bool          `json:"is_request_gateway,omitempty"`
}

type BenchmarkReport struct {
	Results []TestResult
}

func (r *BenchmarkReport) PrintTable() {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Scenario", "Case", "Cost", "IsRequestGateway"})

	for _, res := range r.Results {
		if err := table.Append([]any{
			res.Scenario,
			res.CaseName,
			res.CostTime.String(),
			res.IsRequestGateway,
		}); err != nil {
			log.Errorw("failed to append row to table", zap.Error(err))
		}
	}
	if err := table.Render(); err != nil {
		log.Errorw("failed to render table", zap.Error(err))
	}
}

func (r *BenchmarkReport) PrintJSON() {
	b, _ := json.MarshalIndent(r.Results, "", "  ")
	fmt.Println(string(b))
}

func (r *BenchmarkReport) AddResult(result TestResult) {
	r.Results = append(r.Results, result)
}

func (r *BenchmarkReport) Add(scenario, caseName string, cost time.Duration) {
	r.Results = append(r.Results, TestResult{
		Scenario: scenario,
		CaseName: caseName,
		CostTime: cost,
	})
}
