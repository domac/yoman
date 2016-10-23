package yoman

import (
	"flag"
	"fmt"
	"github.com/domac/yoman/config"
	"github.com/domac/yoman/core"
	"strings"
	"sync"
	"time"
)

var APP_VERSION = "1.0"

const (
	Oid_Inbound  = "1.3.6.1.2.1.31.1.1.1.6"  //入站OID
	Oid_Outbound = "1.3.6.1.2.1.31.1.1.1.10" //出站OID

)

var (
	work_num    = flag.Int("w", 100, "num of worker num")                       //执行的协程数量
	interval    = flag.Int("i", 10, "interval of worker execute")               //任务执行间隔
	timeout     = flag.Int("timeout", 500, "timeout of smmp get data")          //SNMP调用超时
	oids        = flag.String("oids", "", "oids for snmp")                      //oids 数据
	datafile    = flag.String("datafile", "", "datafile for loading snmp data") //数据文件
	datauri     = flag.String("datauri", "", "data uri for getting snmp data")  //数据接口
	Debug       = flag.Bool("debug", false, "debug mode")                       //debug模式
	priority    = flag.Int("pp", 0, "num of priority worker num")               //优先执行个数
	retries     = flag.Int("rt", 0, "num of retries num")
	reporturi   = flag.String("reporturi", "http://localhost:8080/switch/flow", "report uri for sending snmp data to the server")
	app_version = flag.Bool("v", false, "the version of yoman")
)

//执行函数
func Startup() {
	flag.Parse()

	if *app_version {
		println(APP_VERSION)
		return
	}

	itvl := time.Duration(*interval)

	if *oids == "" {
		println("no oids found, please input oid value by `-oids=` ")
		return
	}
	oidlist := strings.Split(*oids, ",")

	var (
		err   error
		wg    sync.WaitGroup
		mpwg  sync.WaitGroup
		items []config.Switch
	)

	if *datauri == "" && *datafile == "" {
		println("no remote uri, please input data interface by `-datauri=` ")
		return
	} else {
		//载入数据优先级: 数据接口 > 数据文件
		if *datauri == "" {
			err = CheckDataFileExist(*datafile)
			if err != nil {
				panic(err)
			}
			items, err = config.LoadSwitchFromFile(*datafile) //从文件获取
		} else {
			//从数据接口中获取数据
			items, err = LoadSwitchFromDataUri(*datauri) //从接口获取
		}
	}

	if err != nil {
		panic(err)
	}

	//创建任务调度器
	d := core.NewDispatcherWithMQ(*work_num, *work_num, &wg, &mpwg)
	d.SetPriority(*priority)

	//设置消息处理方法
	r := NewReport(*reporturi)
	d.SetMF(GenerateMessageReportMethod(r))

	//启动调度器
	d.RunWithLimiter(itvl * time.Millisecond)
	defer d.Stop()

	wg.Add(1)
	mpwg.Add(1)
	start := time.Now()
	go func() {
		for i, oid := range oidlist {
			for j, item := range items {
				id := fmt.Sprintf("%d-%d", i, j)
				job := NewJob(id, item.Host, item.Community, oid, *timeout, *retries)
				t := core.CreateTask(job, "Do")
				d.SubmitTask(t)
			}
		}
		fmt.Println("任务派分完成,正在执行中...")
		wg.Done()
		mpwg.Done()
	}()
	wg.Wait()
	mpwg.Wait()

	cost := fmt.Sprintf("%v", time.Now().Sub(start).Seconds())

	fmt.Println("任务执行完成,正在整理采集数据进行上报...")
	//数据上报
	r_start := time.Now()
	sdc := r.SendData()
	r_cost := fmt.Sprintf("%v", time.Now().Sub(r_start).Seconds())

	fmt.Println("数据上报完成!")

	fmt.Println("\n")
	fmt.Printf("----------- 全部处理完成 : %s ----------- \n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("# 执行snmp请求协程数量 : %d\n", Wgroutinue)
	fmt.Printf("# 执行snmp错误数量 : %d\n", Erroc)
	fmt.Printf("# 上报调用次数 : %d\n", len(r.Data))
	fmt.Printf("# 上报数据批次数量 : %d\n", sdc)
	fmt.Printf("# Snmp采集耗时 (秒) : %v\n", cost)
	fmt.Printf("# 数据上报耗时 (秒): %v\n", r_cost)
	fmt.Println("\n")
}
