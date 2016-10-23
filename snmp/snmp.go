package snmp

import (
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"time"
)

type WapSNMP struct {
	Target    string
	Community string
	Version   SNMPVersion
	timeout   time.Duration
	retries   int
	conn      net.Conn
}

type SNMPValue struct {
	Oid   Oid
	Value interface{}
}

const (
	bufSize int = 16384
)

//创建SNMP客户端
func NewWapSNMP(target, community string, version SNMPVersion, timeout time.Duration, retries int) (*WapSNMP, error) {
	targetPort := fmt.Sprintf("%s:161", target)
	conn, err := net.DialTimeout("udp", targetPort, timeout)
	if err != nil {
		return nil, fmt.Errorf(`error connecting to ("udp", "%s"): %s`, targetPort, err)
	}
	return &WapSNMP{target, community, version, timeout, retries, conn}, nil
}

//创建自定义连接的SNMP客户端
func NewWapSNMPOnConn(target, community string, version SNMPVersion, timeout time.Duration, retries int, conn net.Conn) *WapSNMP {
	return &WapSNMP{target, community, version, timeout, retries, conn}
}

//生成随机的请求ID
func RandomRequestID() int {
	return int(rand.Int31())
}

//轮训请求
func poll(conn net.Conn, toSend []byte, respondBuffer []byte, retries int, timeout time.Duration) (int, error) {
	var err error
	for i := 0; i < retries+1; i++ {

		if i >= 1 {
			fmt.Printf("正在进行snmp的第%d次重试.\n", i)
		}

		deadline := time.Now().Add(timeout)

		if err = conn.SetWriteDeadline(deadline); err != nil {
			fmt.Errorf("Couldn't set write deadline. Retrying. Retry %d/%d\n", i, retries)
			continue
		}
		if _, err = conn.Write(toSend); err != nil {
			fmt.Errorf("Couldn't write. Retrying. Retry %d/%d\n", i, retries)
			continue
		}
		//超时
		deadline = time.Now().Add(timeout)
		if err = conn.SetReadDeadline(deadline); err != nil {
			fmt.Errorf("Couldn't set read deadline. Retrying. Retry %d/%d\n", i, retries)
			continue
		}

		numRead := 0
		if numRead, err = conn.Read(respondBuffer); err != nil {
			fmt.Errorf("Couldn't read. Retrying. Retry %d/%d\n", i, retries)
			continue
		}

		return numRead, nil
	}
	return 0, err
}

//请求获取
func (w WapSNMP) Get(oid Oid) (interface{}, error) {
	requestID := RandomRequestID()
	req, err := EncodeSequence([]interface{}{Sequence, int(w.Version), w.Community,
		[]interface{}{AsnGetRequest, requestID, 0, 0,
			[]interface{}{Sequence,
				[]interface{}{Sequence, oid, nil}}}})
	if err != nil {
		return nil, err
	}

	response := make([]byte, bufSize, bufSize)
	numRead, err := poll(w.conn, req, response, w.retries, w.timeout)
	if err != nil {
		return nil, err
	}

	decodedResponse, err := DecodeSequence(response[:numRead])
	if err != nil {
		return nil, err
	}

	respPacket := decodedResponse[3].([]interface{})
	varbinds := respPacket[4].([]interface{})
	result := varbinds[1].([]interface{})[2]

	return result, nil
}

//接收多个OID的合并处理
func (w WapSNMP) GetMultiple(oids []Oid) (map[string]interface{}, error) {
	requestID := RandomRequestID()

	varbinds := []interface{}{Sequence}
	for _, oid := range oids {
		varbinds = append(varbinds, []interface{}{Sequence, oid, nil})
	}
	req, err := EncodeSequence([]interface{}{Sequence, int(w.Version), w.Community,
		[]interface{}{AsnGetRequest, requestID, 0, 0, varbinds}})

	if err != nil {
		return nil, err
	}

	response := make([]byte, bufSize, bufSize)
	numRead, err := poll(w.conn, req, response, w.retries, w.timeout)
	if err != nil {
		return nil, err
	}

	decodedResponse, err := DecodeSequence(response[:numRead])
	if err != nil {
		return nil, err
	}

	respPacket := decodedResponse[3].([]interface{})
	respVarbinds := respPacket[4].([]interface{})

	result := make(map[string]interface{})
	for _, v := range respVarbinds[1:] { // First element is just a sequence
		oid := v.([]interface{})[1].(Oid).String()
		value := v.([]interface{})[2]
		result[oid] = value
	}

	return result, nil
}

func (w WapSNMP) Set(oid Oid, value interface{}) (interface{}, error) {
	requestID := RandomRequestID()
	req, err := EncodeSequence([]interface{}{Sequence, int(w.Version), w.Community,
		[]interface{}{AsnSetRequest, requestID, 0, 0,
			[]interface{}{Sequence,
				[]interface{}{Sequence, oid, value}}}})
	if err != nil {
		return nil, err
	}

	response := make([]byte, bufSize, bufSize)
	numRead, err := poll(w.conn, req, response, w.retries, w.timeout)
	if err != nil {
		return nil, err
	}

	decodedResponse, err := DecodeSequence(response[:numRead])
	if err != nil {
		return nil, err
	}

	respPacket := decodedResponse[3].([]interface{})
	varbinds := respPacket[4].([]interface{})
	result := varbinds[1].([]interface{})[2]

	return result, nil
}

func (w WapSNMP) SetMultiple(toset map[string]interface{}) (map[string]interface{}, error) {
	requestID := RandomRequestID()

	varbinds := []interface{}{Sequence}
	for oid, value := range toset {
		varbinds = append(varbinds, []interface{}{Sequence, oid, value})
	}
	req, err := EncodeSequence([]interface{}{Sequence, int(w.Version), w.Community,
		[]interface{}{AsnGetRequest, requestID, 0, 0, varbinds}})

	if err != nil {
		return nil, err
	}

	response := make([]byte, bufSize, bufSize)
	numRead, err := poll(w.conn, req, response, w.retries, w.timeout)
	if err != nil {
		return nil, err
	}

	decodedResponse, err := DecodeSequence(response[:numRead])
	if err != nil {
		return nil, err
	}

	respPacket := decodedResponse[3].([]interface{})
	respVarbinds := respPacket[4].([]interface{})

	result := make(map[string]interface{})
	for _, v := range respVarbinds[1:] { // First element is just a sequence
		oid := v.([]interface{})[1].(Oid).String()
		value := v.([]interface{})[2]
		result[oid] = value
	}

	return result, nil
}

//获取下一个
func (w WapSNMP) GetNext(oid Oid) (*Oid, interface{}, error) {
	requestID := RandomRequestID()
	req, err := EncodeSequence([]interface{}{Sequence, int(w.Version), w.Community,
		[]interface{}{AsnGetNextRequest, requestID, 0, 0,
			[]interface{}{Sequence,
				[]interface{}{Sequence, oid, nil}}}})
	if err != nil {
		return nil, nil, err
	}

	response := make([]byte, bufSize)
	numRead, err := poll(w.conn, req, response, w.retries, w.timeout)
	if err != nil {
		return nil, nil, err
	}

	decodedResponse, err := DecodeSequence(response[:numRead])
	if err != nil {
		return nil, nil, err
	}

	respPacket := decodedResponse[3].([]interface{})
	varbinds := respPacket[4].([]interface{})
	result := varbinds[1].([]interface{})

	resultOid := result[1].(Oid)
	resultVal := result[2]

	return &resultOid, resultVal, nil
}

func (w WapSNMP) GetBulk(oid Oid, maxRepetitions int) (map[string]interface{}, error) {
	requestID := RandomRequestID()
	req, err := EncodeSequence([]interface{}{Sequence, int(w.Version), w.Community,
		[]interface{}{AsnGetBulkRequest, requestID, 0, maxRepetitions,
			[]interface{}{Sequence,
				[]interface{}{Sequence, oid, nil}}}})
	if err != nil {
		return nil, err
	}

	response := make([]byte, bufSize, bufSize)
	numRead, err := poll(w.conn, req, response, w.retries, w.timeout)
	if err != nil {
		return nil, err
	}

	decodedResponse, err := DecodeSequence(response[:numRead])
	if err != nil {
		return nil, err
	}

	respPacket := decodedResponse[3].([]interface{})
	respVarbinds := respPacket[4].([]interface{})

	result := make(map[string]interface{})
	for _, v := range respVarbinds[1:] {
		oid := v.([]interface{})[1].(Oid).String()
		value := v.([]interface{})[2]
		result[oid] = value
	}

	return result, nil
}

func (w WapSNMP) GetBulkArray(oid Oid, maxRepetitions int) ([]SNMPValue, error) {
	requestID := RandomRequestID()
	req, err := EncodeSequence([]interface{}{Sequence, int(w.Version), w.Community,
		[]interface{}{AsnGetBulkRequest, requestID, 0, maxRepetitions,
			[]interface{}{Sequence,
				[]interface{}{Sequence, oid, nil}}}})
	if err != nil {
		return nil, err
	}

	response := make([]byte, bufSize, bufSize)
	numRead, err := poll(w.conn, req, response, w.retries, w.timeout)
	if err != nil {
		return nil, err
	}

	decodedResponse, err := DecodeSequence(response[:numRead])
	if err != nil {
		return nil, fmt.Errorf("error during sequence decoding: %v", err)
	}

	respPacket := decodedResponse[3].([]interface{})
	respVarbinds := respPacket[4].([]interface{})

	result := make([]SNMPValue, 0, len(respVarbinds[1:]))
	for _, v := range respVarbinds[1:] { // First element is just a sequence
		oid := v.([]interface{})[1].(Oid)
		value := v.([]interface{})[2]
		result = append(result, SNMPValue{oid, value})
	}

	return result, nil
}

//请求的结果形成Table的形式并返回
func (w WapSNMP) GetTable(oid Oid) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	lastOid := oid.Copy()
	for lastOid.Within(oid) {
		results, err := w.GetBulkArray(lastOid, 50)
		if err != nil {
			//return nil, fmt.Errorf("received GetBulk error => %v\n", err)
			return nil, fmt.Errorf("oid(%s) received GetBulk error => %v\n", lastOid.String(), err)
		}
		newLastOid := lastOid.Copy()
		for _, v := range results {
			if v.Oid.Within(oid) {
				result[v.Oid.String()] = v.Value
			}
			newLastOid = v.Oid
		}

		if reflect.DeepEqual(lastOid, newLastOid) {
			break
		}
		lastOid = newLastOid
	}
	return result, nil
}

func (w WapSNMP) Close() error {
	return w.conn.Close()
}
