package yoman

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
)

//上报数据属性
type Property struct {
	Oid         string `json:"oid"`
	Inbound     int64  `json:"in_bound"`
	OutBound    int64  `json:"out_bound"`
	Port        int    `json:"port"`
	Host        string `json:"host"`
	Start_clock int64  `json:"start_clock"`
	Clock       int64  `json:"clock"`
}

type Report struct {
	mutex     sync.Mutex //互斥锁
	Data      map[string][]*SwitchResult
	Reporturi string
}

func NewReport(reporturi string) *Report {
	return &Report{
		Data:      make(map[string][]*SwitchResult),
		Reporturi: reporturi}
}

func (r *Report) AddResult(s *SwitchResult) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.Data[s.Shost]; ok {
		r.Data[s.Shost] = append(r.Data[s.Shost], s)
	} else {
		list := []*SwitchResult{}
		r.Data[s.Shost] = append(list, s)
	}
}

//统计原始上报数据的数据行数
func (r *Report) GetReportCount() int64 {
	var count int64 = 0
	for _, v := range r.Data {
		for range v {
			count = count + 1
		}
	}
	return count
}

//发送报告数据
func (r *Report) SendData() int64 {
	count := int64(0)
	for _, v := range r.Data {
		c := int64(0)
		params := make(map[string]string)
		params["data"], c = SwitchCollectData(v) //每个交换机的采集结果
		count = count + c
		if r.Reporturi != "" && len(r.Reporturi) > 5 {
			yomanClient.Post(r.Reporturi, params)
		}
	}
	return count
}

//生成端口为参考指标的上报数据
func SwitchCollectData(switchResults []*SwitchResult) (string, int64) {
	portMap := make(map[string]*Property) //port-oid-data
	for _, sr := range switchResults {
		property := new(Property)
		if _, ok := portMap[sr.SPort]; ok {
			property = portMap[sr.SPort]
		}
		property.setProperty(sr)
		portMap[sr.SPort] = property
	}
	return ConvertToJson(portMap)
}

//设置属性
func (p *Property) setProperty(sr *SwitchResult) {

	p.Start_clock = sr.STime
	p.Clock = sr.ETime
	p.Host = sr.Shost
	p.Oid = sr.Oid
	flow, _ := strconv.Atoi(sr.SFlow)

	if !*Debug {
		if sr.Oid == Oid_Inbound {
			p.Inbound = int64(flow)
		} else if sr.Oid == Oid_Outbound {
			p.OutBound = int64(flow)
		}
	} else {
		if strings.Contains(sr.Oid, Oid_Inbound) {
			p.Inbound = int64(flow)
		} else if strings.Contains(sr.Oid, Oid_Outbound) {
			p.OutBound = int64(flow)
		}
	}

}

//转化为json格式
func ConvertToJson(p map[string]*Property) (data string, count int64) {
	if p == nil {
		return data, int64(0)
	}

	plist := []*Property{}

	for port, property := range p {
		count++
		pp, _ := strconv.Atoi(port)
		property.Port = pp
		plist = append(plist, property)
	}

	if b, err := json.Marshal(plist); err == nil {
		data = string(b)
	}
	return data, count
}
