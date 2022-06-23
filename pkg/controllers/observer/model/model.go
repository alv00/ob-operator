/*
Copyright (c) 2021 OceanBase
ob-operator is licensed under Mulan PSL v2.
You can use this software according to the terms and conditions of the Mulan PSL v2.
You may obtain a copy of Mulan PSL v2 at:
         http://license.coscl.org.cn/MulanPSL2
THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
See the Mulan PSL v2 for more details.
*/

package model

type AllServer struct {
	ID              int64
	Zone            string
	SvrIP           string
	SvrPort         int64
	InnerPort       int64
	WithRootService int64
	WithPartition   int64
	Status          string
    StartServiceTime int64
}

type AllVirtualCoreMeta struct {
	Zone         string
	SvrIP        string
	SvrPort      int64
	Role         int64
	PartitionIP  int64
	PartitionCnt int64
}

type RSJobStatus struct {
	JobStatus  string
	ReturnCode int64
	Progress   int64
}

type OBAgent struct {
	ID   int64
	Zone string
}
