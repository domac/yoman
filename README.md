# yoman

#### 基于SNMP的交换机出入站流量采集工具


#### 如何使用 ?


1.创建项目GOPATH路径

```sh
   go get -u -v https://github.com/domac/yoman.git
```

2.运行编译脚本

```sh
    $ cd $GOPATH/src/github.com/domac/yoman
    
    $ make build   //若构建成功,会在代码目录生产 releases 文件夹
```

3.执行静态链接文件

```sh
$ cd $GOPATH/src/github.com/domac/yoman/releases 

$ ./yoman -datafile=/your/datafile/path -w 10000 -i 10 -timeout 500 -oids=1.3.6.1.2.1.31.1.1.1.6
```

4.工具使用参考

```sh

自定义交换机数据本地文件:
$ ./yoman -datafile=/your/datafile/path -w 5000 -i 10 -timeout 10000 -oids=1.3.6.1.2.1.31.1.1.1.6 -rt 5

自定义交换机数据接口:
$ ./yoman -datauri=http://your_data_webservice/list -w 5000 -i 10 -timeout 10000 -oids=1.3.6.1.2.1.31.1.1.1.6 -rt 5


```


5.参数说明

```
    > w (必填): 最大工作groutinue并发任务数 (一般可以设置相对大的整数,但也不是越大越好,根据实际需要设置)

    > i : 工作groutinue分发间隔

    > pp : 优先执行数(默认为0 : 免分发间隔影响的groutinue数量) //慎用

    > timeout (必填): snmp连接请求超时时间/毫秒 (设置过小的话,容易出现连接snmp i/o timeout,根据实际情况设置)
    
    > rt (建议): snmp连接失败重试次数

    > oids (必填): snmp oid, 多个以逗号分隔开

    > datafile : 交换机数据文件所在路径: 文件内容格式为 `[{"host": "1.1.1.1", "community": "public"} ...]`
    
    > datauri : 与datafile参数类似,表示交换机数据获取的web接口: (例如: http://switchserver/switchs/list.do) 

    > reporturi : 自定义的上报接口      

    > v : 输出版本信息                                                                                                                                                   

```


6.依赖管理 （可忽略）

yoman 使用`godep`工具进行第三方包的管理。

为了保持统一，使用第三方包编写功能的后，请使用 `godep save` 命令执行进行依赖管理。

 - 若要把第三方包load到本地，可使用 `godep reload`

 - 若要本地测试构建，请使用命令 `godep go build main.go`