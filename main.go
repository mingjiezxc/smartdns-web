package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"

	_ "smartdns-web/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	ctx       = context.Background()
	cli       *clientv3.Client
	appConfig ConfigYaml
)

type ConfigYaml struct {
	EtcdServers  []string `yaml:"EtcdAddr"`
	EtcdUser     string   `yaml:"EtcdUser"`
	EtcdPassword string   `yaml:"EtcdPassword"`
	ServerAddr   string   `yaml:"ServerAddr"`
}

func init() {

	// read config file
	configfile, err := ioutil.ReadFile("./config.yaml")
	if ErrCheck(err) {
		os.Exit(1)
	}

	// yaml marshal config
	err = yaml.Unmarshal(configfile, &appConfig)
	if ErrCheck(err) {
		os.Exit(2)
	}

	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   appConfig.EtcdServers,
		Username:    appConfig.EtcdUser,
		Password:    appConfig.EtcdPassword,
		DialTimeout: 10 * time.Second,
	})
	if ErrCheck(err) {
		os.Exit(3)
	}

}

// @title Ping Agnet Manager Api
// @version 1.0
// @description This is a sample server Petstore server.
// @termsOfService https://github.com/mingjiezxc

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath /v1
func main() {

	// read config file
	configfile, err := ioutil.ReadFile("./config.yaml")
	if ErrCheck(err) {
		os.Exit(1)
	}

	// yaml marshal config
	err = yaml.Unmarshal(configfile, &appConfig)
	if ErrCheck(err) {
		os.Exit(2)
	}

	r := gin.Default()
	// 自定义分隔符
	// r.Delims("{[{", "}]}")
	// 配置模板
	r.LoadHTMLGlob("web/*")
	// 配置静态文件夹路径 第一个参数是api，第二个是文件夹路径
	// r.StaticFS("/static", http.Dir("./AdminLTE"))

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"tableUrl": "/v1/smartdns",
		})
	})

	r.GET("/smartdns", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"tableUrl": "/v1/smartdns",
		})
	})

	r.GET("/linedns", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"tableUrl": "/v1/linedns",
		})
	})

	r.GET("/acl", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"tableUrl":    "/v1/acl/ip/cidr",
			"allowerEdit": true,
		})
	})

	r.GET("/forward", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"tableUrl":    "/v1/forward/groups",
			"allowerEdit": true,
		})
	})

	r.GET("/test", func(c *gin.Context) {
		c.HTML(http.StatusOK, "test.html", gin.H{})
	})

	v1Group := r.Group("/v1")
	v1Group.GET("/ping", Ping)
	v1Group.GET("/smartdns", GetSmartDns)
	v1Group.GET("/linedns", GetLineDns)

	// acl ip cidr
	v1Group.GET("/acl/ip/cidr", GetAclIpAllCidr)
	v1Group.POST("/acl/ip/cidr", PostAclIpCidr)
	v1Group.GET("/acl/ip/cidr/:network/:netmask", GetAclIpCidr)
	v1Group.DELETE("/acl/ip/cidr/:network/:netmask", DelAclIpCidr)

	v1Group.GET("/acl/ip/pool", GetAclIpPool)

	// forward
	v1Group.GET("/forward/groups", GetForwardGroup)
	v1Group.POST("/forward/group", PostForwardGroup)
	v1Group.GET("/forward/group/:group", GetForwardGroupInfo)
	v1Group.DELETE("/forward/group/:group", DelForwardGroup)

	// forward domain
	v1Group.GET("/forward/group/:group/:domain", GetForwardGroupDomain)
	// v1Group.POST("/forward/group/:group/:domain", PostForwardGroupDomain)
	v1Group.DELETE("/forward/group/:group/:domain", DelForwardGroupDomain)

	url := ginSwagger.URL("/swagger/doc.json")
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	r.Run(appConfig.ServerAddr) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

// @Summary a ping api
// @Description ping
// @Accept  text/plain
// @Success 200 {strng} string	"pong"
// @Router /ping [get]
func Ping(c *gin.Context) {
	c.String(200, "pong")
}

type BaseReturn struct {
	Status bool
	Data   interface{}
}

func ErrCheck(err error) bool {
	if err != nil {
		log.Println(err.Error())
		return true
	}
	return false
}

func GetTableColumn(data interface{}) (listData []map[string]string) {

	val := reflect.ValueOf(data)
	for i := 0; i < val.Type().NumField(); i++ {
		tmpData := make(map[string]string)
		tmpLabel := val.Type().Field(i).Tag.Get("label")
		if tmpLabel == "" {
			continue
		}
		tmpData["label"] = tmpLabel
		tmpData["prop"] = val.Type().Field(i).Tag.Get("json")
		tmpData["width"] = val.Type().Field(i).Tag.Get("width")
		listData = append(listData, tmpData)
	}
	return

}

type ReturnTable struct {
	Data   interface{} `json:"data"`
	Column interface{} `json:"column"`
}

type SmartdnsStatus struct {
	Name   string `json:"name" label:"名称" width:"100"`
	Status string `json:"status" label:"状态"  width:"100"`
	Lease  int64  `json:"lease" label:"租约" width:"100"`
}

func GetSmartDns(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := cli.Get(ctx, "/smartdns/app/", clientv3.WithPrefix())
	defer cancel()
	ErrCheck(err)

	var smartdnsList []SmartdnsStatus

	for _, ev := range resp.Kvs {
		name := strings.Split(string(ev.Key), "/")[3]
		tmpStatus := SmartdnsStatus{
			Name:   name,
			Status: "online",
			Lease:  ev.Lease,
		}
		smartdnsList = append(smartdnsList, tmpStatus)

	}

	c.JSON(200, ReturnTable{
		Data:   smartdnsList,
		Column: GetTableColumn(SmartdnsStatus{}),
	})

}

type LineDnsStatus struct {
	ZoneName string `json:"zoneName" label:"区域" width:"100"`
	LineType string `json:"lineType" label:"线路"  width:"100"`
	Addr     string `json:"addr" label:"地址"  width:"150"`
	Lease    int64  `json:"lease" label:"租约"  width:"100"`
}

func GetLineDns(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := cli.Get(ctx, "/line/dns/", clientv3.WithPrefix())
	defer cancel()
	ErrCheck(err)

	var tmpStatusList []LineDnsStatus

	for _, ev := range resp.Kvs {
		tmpList := strings.Split(string(ev.Key), "/")

		tmpStatus := LineDnsStatus{
			ZoneName: tmpList[3],
			LineType: tmpList[4],
			Addr:     tmpList[5],
			Lease:    ev.Lease,
		}
		tmpStatusList = append(tmpStatusList, tmpStatus)

	}

	c.JSON(200, ReturnTable{
		Data:   tmpStatusList,
		Column: GetTableColumn(LineDnsStatus{}),
	})
}

type AclStatus struct {
	IP                 string   `json:"ip" label:"IP" width:"180"`
	Cidr               string   `json:"cidr" label:"CIDR" width:"180"`
	Netmask            int64    `json:"netmask"`
	MasterLineDnsReStr string   `json:"masterLineDnsReStr" label:"主要线路" width:"100"`
	MasterDns          []string `json:"masterDns" label:"主要NDS" width:"140"`
	BackupLineDnsReStr string   `json:"backupLineDnsReStr" label:"备用线路" width:"100"`
	BackupDns          []string `json:"backupDns" label:"备用DNS" width:"140"`
	ForwardGroup       []string `json:"forwardGroup" label:"转发组" width:"100"`
	Timeout            int64    `json:"timeout"`

	ID          int    `json:"id"`
	HasChildren bool   `json:"hasChildren"`
	DataUrl     string `json:"dataUrl"`
	UpdateUrl   string `json:"updateUrl"`
	DelUrl      string `json:"DelUrl"`
}

func GetAclIpAllCidr(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := cli.Get(ctx, "/acl/ip/cidr/", clientv3.WithPrefix())
	defer cancel()
	ErrCheck(err)

	var tmpStatusList []AclStatus

	for id, ev := range resp.Kvs {

		var tmpStatus AclStatus
		err = json.Unmarshal(ev.Value, &tmpStatus)
		tmpStatus.HasChildren = true
		tmpStatus.ID = id
		tmpStatus.IP = tmpStatus.Cidr
		tmpStatus.DataUrl = "/v1/acl/ip/cidr/" + tmpStatus.Cidr
		tmpStatus.UpdateUrl = "/v1/acl/ip/cidr"
		tmpStatus.DelUrl = "/v1/acl/ip/cidr/" + tmpStatus.Cidr

		if !ErrCheck(err) {
			tmpStatusList = append(tmpStatusList, tmpStatus)
		}

	}

	if len(tmpStatusList) == 0 {
		tmpStatusList = append(tmpStatusList, AclStatus{
			UpdateUrl: "/v1/acl/ip/cidr",
		})
	}

	c.JSON(200, ReturnTable{
		Data:   tmpStatusList,
		Column: GetTableColumn(AclStatus{}),
	})
}

func PostAclIpCidr(c *gin.Context) {
	var data AclStatus
	if err := c.ShouldBindJSON(&data); ErrCheck(err) {
		c.JSON(501, err)
		return
	}

	ips, _, err := Hosts(data.Cidr)
	if ErrCheck(err) {
		c.JSON(502, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	data.IP = data.Cidr
	dataJson, _ := json.Marshal(data)
	dataJsonStr := string(dataJson)
	_, err = cli.Put(ctx, "/acl/ip/cidr/"+data.Cidr, dataJsonStr)
	if ErrCheck(err) {
		c.JSON(503, err)
		return
	}

	for _, ip := range ips {
		var tmpData AclStatus
		resp, err := cli.Get(ctx, "/acl/ip/pool/"+ip)
		if err != nil {
			continue
		}

		if len(resp.Kvs) >= 1 {
			err = json.Unmarshal(resp.Kvs[0].Value, &tmpData)
			if err != nil {
				continue
			}
		}

		if tmpData.Netmask > data.Netmask {
			continue
		}

		data.IP = ip
		tmpDataJson, _ := json.Marshal(data)

		_, err = cli.Put(ctx, "/acl/ip/pool/"+ip, string(tmpDataJson))
		ErrCheck(err)

	}

	c.JSON(200, `{"mesg":"update done"}`)
}

func GetAclIpPool(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := cli.Get(ctx, "/acl/ip/pool/", clientv3.WithPrefix())
	defer cancel()
	ErrCheck(err)

	var tmpStatusList []AclStatus

	for _, ev := range resp.Kvs {
		var tmpStatus AclStatus
		err = json.Unmarshal(ev.Value, &tmpStatus)

		if !ErrCheck(err) {
			tmpStatusList = append(tmpStatusList, tmpStatus)
		}

	}

	c.JSON(200, ReturnTable{
		Data:   tmpStatusList,
		Column: GetTableColumn(AclStatus{}),
	})
}

func GetAclIpCidr(c *gin.Context) {
	network := c.Param("network")
	netmask := c.Param("netmask")

	ips, count, err := Hosts(network + "/" + netmask)
	if err != nil {
		c.JSON(501, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tmpStatusList []AclStatus
	for n := 0; n < count; n++ {
		resp, err := cli.Get(ctx, "/acl/ip/pool/"+ips[n])
		if err != nil {
			continue
		}
		for _, ev := range resp.Kvs {
			var tmpStatus AclStatus

			err = json.Unmarshal(ev.Value, &tmpStatus)
			tmpStatus.UpdateUrl = "/v1/acl/ip/cidr"
			tmpStatus.DelUrl = "/v1/acl/ip/cidr/" + tmpStatus.Cidr

			if err != nil {
				continue
			}
			tmpStatus.ID = n + 1000
			tmpStatus.HasChildren = false
			tmpStatusList = append(tmpStatusList, tmpStatus)
		}

	}

	c.JSON(200, ReturnTable{
		Data:   tmpStatusList,
		Column: GetTableColumn(AclStatus{}),
	})
}

func DelAclIpCidr(c *gin.Context) {
	network := c.Param("network")
	netmask := c.Param("netmask")

	ips, count, err := Hosts(network + "/" + netmask)
	if err != nil {
		c.JSON(501, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tmpStatusList []AclStatus
	for n := 0; n < count; n++ {
		_, err := cli.Delete(ctx, "/acl/ip/pool/"+ips[n])
		if err != nil {
			continue
		}
	}
	_, err = cli.Delete(ctx, "/v1/acl/ip/cidr/"+network+"/"+netmask)
	if ErrCheck(err) {
		c.JSON(503, err)
	}

	c.JSON(200, ReturnTable{
		Data:   tmpStatusList,
		Column: GetTableColumn(AclStatus{}),
	})
}

type ForwardStatus struct {
	GroupName    string   `json:"groupName" label:"组名称" width:"150"`
	Domain       string   `json:"domain" label:"域名" width:"150"`
	LineDnsReStr string   `json:"lineDnsReStr" label:"线路文本" width:"150"`
	Dns          []string `json:"dns" label:"DNS"  width:"250"`

	ID          int    `json:"id"`
	HasChildren bool   `json:"hasChildren"`
	DataUrl     string `json:"dataUrl"`
	UpdateUrl   string `json:"updateUrl"`
	DelUrl      string `json:"delUrl"`
}

func GetForwardGroup(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := cli.Get(ctx, "/forward/groups/", clientv3.WithPrefix())
	defer cancel()
	ErrCheck(err)

	var tmpStatusList []ForwardStatus

	for id, ev := range resp.Kvs {

		tmpList := strings.Split(string(ev.Key), "/")

		var tmpStatus ForwardStatus
		tmpStatus.GroupName = tmpList[3]
		tmpStatus.HasChildren = true
		tmpStatus.ID = id + 20000
		tmpStatus.DataUrl = "/v1/forward/group/" + tmpStatus.GroupName
		tmpStatus.UpdateUrl = "/v1/forward/group"
		tmpStatus.DelUrl = "/v1/forward/group/" + tmpStatus.GroupName

		if !ErrCheck(err) {
			tmpStatusList = append(tmpStatusList, tmpStatus)
		}

	}

	if len(tmpStatusList) == 0 {
		tmpStatusList = append(tmpStatusList, ForwardStatus{
			Dns:       []string{""},
			UpdateUrl: "/v1/forward/group",
		})
	}

	c.JSON(200, ReturnTable{
		Data:   tmpStatusList,
		Column: GetTableColumn(ForwardStatus{}),
	})
}

func PostForwardGroup(c *gin.Context) {
	var data ForwardStatus
	if err := c.ShouldBindJSON(&data); ErrCheck(err) {
		c.JSON(501, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := cli.Get(ctx, "/forward/groups/"+data.GroupName)

	if err == nil && len(resp.Kvs) == 0 {
		_, err := cli.Put(ctx, "/forward/groups/"+data.GroupName, "ok")
		if ErrCheck(err) {
			c.JSON(502, err)
			return
		}
	}

	dataJson, err := json.Marshal(data)
	if ErrCheck(err) {
		c.JSON(503, err)
		return
	}

	_, err = cli.Put(ctx, "/forward/group/"+data.GroupName+"/"+data.Domain, string(dataJson))
	if ErrCheck(err) {
		c.JSON(504, err)
		return
	}

	c.JSON(200, `{"mesg":"update done"}`)
}

func GetForwardGroupInfo(c *gin.Context) {

	group := c.Param("group")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := cli.Get(ctx, "/forward/group/"+group+"/", clientv3.WithPrefix())
	defer cancel()
	ErrCheck(err)

	var tmpStatusList []ForwardStatus

	for id, ev := range resp.Kvs {

		// tmpList := strings.Split(string(ev.Key), "/")

		var tmpStatus ForwardStatus
		err = json.Unmarshal(ev.Value, &tmpStatus)
		// tmpStatus.Domain = tmpList[4]
		// tmpStatus.GroupName = tmpList[3]

		tmpStatus.HasChildren = false
		tmpStatus.ID = id + 30000
		tmpStatus.UpdateUrl = "/v1/forward/group"
		tmpStatus.DelUrl = "/v1/forward/group/" + tmpStatus.GroupName + "/" + tmpStatus.Domain

		if !ErrCheck(err) {
			tmpStatusList = append(tmpStatusList, tmpStatus)
		}

	}

	c.JSON(200, ReturnTable{
		Data:   tmpStatusList,
		Column: GetTableColumn(ForwardStatus{}),
	})
}

func DelForwardGroup(c *gin.Context) {
	group := c.Param("group")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := cli.Delete(ctx, "/forward/group/"+group+"/", clientv3.WithPrefix())
	if ErrCheck(err) {
		c.JSON(501, err)
		return
	}

	_, err = cli.Delete(ctx, "/forward/group/"+group)
	if ErrCheck(err) {
		c.JSON(502, err)
		return
	}

	ErrCheck(err)

	c.JSON(200, `{"mesg":"update done"}`)
}

func GetForwardGroupDomain(c *gin.Context) {
	group := c.Param("group")
	domain := c.Param("domain")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := cli.Get(ctx, "/forward/group/"+group+"/"+domain)
	if ErrCheck(err) {
		c.JSON(501, err)
		return
	}

	var tmpStatusList []ForwardStatus

	for id, ev := range resp.Kvs {

		var tmpStatus ForwardStatus
		err = json.Unmarshal(ev.Value, &tmpStatus)

		tmpStatus.HasChildren = false
		tmpStatus.ID = id + 40000
		tmpStatus.UpdateUrl = "/forward/group"
		tmpStatus.DelUrl = "/forward/group/" + tmpStatus.GroupName + "/" + tmpStatus.Domain

		if !ErrCheck(err) {
			tmpStatusList = append(tmpStatusList, tmpStatus)
		}

	}

	c.JSON(200, ReturnTable{
		Data:   tmpStatusList,
		Column: GetTableColumn(ForwardStatus{}),
	})
}

func PostForwardGroupDomain(c *gin.Context) {

	var data ForwardStatus
	if err := c.ShouldBindJSON(&data); ErrCheck(err) {
		c.JSON(501, err)
		return
	}

	dataJson, _ := json.Marshal(data)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := cli.Put(ctx, "/forward/group/"+data.GroupName+"/"+data.Domain, string(dataJson))
	if ErrCheck(err) {
		c.JSON(501, err)
		return
	}

	c.JSON(200, `{"mesg":"update done"}`)
}

func DelForwardGroupDomain(c *gin.Context) {
	group := c.Param("group")
	domain := c.Param("domain")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := cli.Delete(ctx, "/forward/group/"+group+"/"+domain)
	log.Println(resp, err)
	if ErrCheck(err) {
		c.JSON(501, err)
		return
	}
	c.JSON(200, `{"mesg":"update done"}`)
}

func Hosts(cidr string) ([]string, int, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, 0, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	// remove network address and broadcast address
	lenIPs := len(ips)
	switch {
	case lenIPs < 2:
		return ips, lenIPs, nil

	default:
		return ips[1 : len(ips)-1], lenIPs - 2, nil
	}
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
