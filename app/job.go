package yoman

import (
	"encoding/json"
	"fmt"
	"github.com/domac/yoman/config"
	"github.com/domac/yoman/core"
	"github.com/domac/yoman/snmp"
	"time"
)

const version = snmp.SNMPv2c //SNMP协议版本
var Erroc int = 0
var Wgroutinue int = 0

//交换机采集结果
type SwitchResult struct {
	Shost string
	SPort string
	SFlow string
	STime int64
	ETime int64
	Oid   string
}

func NewSwitchResult(host, port, flow, oid string) *SwitchResult {
	return &SwitchResult{
		Shost: host,
		SPort: port,
		SFlow: flow,
		Oid:   oid,
	}
}

//交互机请求信息
type SwitchRequest struct {
	Message string          `json:"message"`
	Code    int             `json:"code"`
	Success bool            `json:"success"`
	Object  []config.Switch `json:"object"`
}

//任务作业结构
type Job struct {
	Id          string
	Host        string
	Community   string
	Oid         string
	Result      []*SwitchResult
	fail        bool
	failMessage string
	Timeout     int
	Retries     int
}

func NewJob(id string, host string, community string, oid string, timeout int, retries int) *Job {
	return &Job{
		Id:        id,
		Oid:       oid,
		Host:      host,
		Community: community,
		Timeout:   timeout,
		Retries:   retries,
	}
}

func (j *Job) SetFailure(message string) {
	j.fail = true
	j.failMessage = message
}

func (j *Job) Do() {
	oid := snmp.MustParseOid(j.Oid)
	result := []*SwitchResult{}
	wsnmp, err := snmp.NewWapSNMP(j.Host, j.Community, version, time.Duration(j.Timeout)*time.Millisecond, j.Retries)
	defer wsnmp.Close()
	if err != nil {
		Erroc++
		j.SetFailure(err.Error())
	} else {

		if !*Debug {
			table, err := wsnmp.GetTable(oid)
			if err != nil {
				Erroc++
				j.SetFailure(err.Error())
			} else {
				for k, v := range table {
					_, port := SplitData(k)
					flow := fmt.Sprintf("%v", v)
					result = append(result, NewSwitchResult(j.Host, port, flow, j.Oid))
				}
			}
		} else {
			woid, err := wsnmp.Get(oid)
			if err != nil {
				Erroc++
				j.SetFailure(err.Error())
			} else {
				flow := fmt.Sprintf("%v", woid)
				_, port := SplitData(oid.String())
				result = append(result, NewSwitchResult(j.Host, port, flow, j.Oid))
			}
		}

	}
	Wgroutinue++
	j.Result = result
}

//上报方法回调
func GenerateMessageReportMethod(r *Report) core.MF {
	return func(task core.Task) {
		tj := (*task.TargetObj).(*Job)
		if !tj.fail {
			//遍历结果,进行上报
			for _, res := range tj.Result {
				res.STime = task.StartTime
				res.ETime = task.EndTime
				r.AddResult(res)
			}
		} else {
			fmt.Printf("oid(%s)连接host(%s)出现异常 : %s \n", tj.Oid, tj.Host, tj.failMessage)
		}
	}
}

func LoadSwitchFromDataUri(url string) ([]config.Switch, error) {

	reconnect_time := 1

	resp, err := yomanClient.Get(url, nil)
	if err != nil {
	REC:
		if reconnect_time <= 3 {
			fmt.Printf("正在进行第%d次重连... \n", reconnect_time)
			time.Sleep(1 * time.Second)
			resp, err = yomanClient.Get(url, nil)
			reconnect_time++
			if err != nil {
				goto REC
			}
		} else {
			fmt.Println("远程获取交换机接口连接失败")
			return nil, err
		}
	}
	defer resp.Body.Close()
	var sr = SwitchRequest{}
	var data []byte
	data, err = resp.ReadAll()
	if err != nil {
		return nil, err
	}
	json.Unmarshal(data, &sr)
	sw := sr.Object
	return sw, nil
}
