package consul

import (
	"fmt"
	"net/http"

	"errors"
	"sync"

	"math/rand"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

//健康检查的端口
const CheckPort = 8080

type ConsulMng struct {
	//consul 服务器地址信息
	consulAddr string
	client     *api.Client
	//注册信息
	rgstInfo *RegisterInfo
	//发现监听信息
	watchers map[string]*ConsulWatcher
}

//服务器注册参数
type RegisterInfo struct {
	serverName string
	serverID   string
	serverAddr string
	serverPort int
}

//健康检查的http
func consulCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "consulCheck")
}

func NewConsulMng(consulAddr string) (*ConsulMng, error) {
	config := &api.Config{
		Address: consulAddr,
	}
	client, err := api.NewClient(config)

	if err != nil {
		return nil, err
	}

	mng := &ConsulMng{
		consulAddr: consulAddr,
		client:     client,
		watchers:   make(map[string]*ConsulWatcher),
	}
	return mng, nil
}

//注册服务
func (mng *ConsulMng) RegisterServer(serverName string, serverID string, addr string, port int) error {
	//注册信息保存在内存里面
	rgstInfo := &RegisterInfo{
		serverName: serverName,
		serverID:   serverID,
		serverAddr: addr,
		serverPort: port,
	}
	mng.rgstInfo = rgstInfo

	//注册
	registration := new(api.AgentServiceRegistration)
	registration.Name = serverName
	registration.ID = serverID
	registration.Address = addr
	registration.Port = port
	registration.Check = &api.AgentServiceCheck{
		HTTP:                           fmt.Sprintf("http://%s:%d%s", addr, CheckPort, "/check"),
		Timeout:                        "3s",
		Interval:                       "5s",
		TTL:                            "10s",
		DeregisterCriticalServiceAfter: "30s",
	}
	err := mng.client.Agent().ServiceRegister(registration)
	if err != nil {
		return err
	}

	//设置健康检查
	http.HandleFunc("/check", consulCheck)
	http.ListenAndServe(fmt.Sprintf(":%d", CheckPort), nil)
	return nil
}

//添加服务发现监听
func (mng *ConsulMng) AddDiscoverWatcher(serverName string, watcher *ConsulWatcher) {
	mng.watchers[serverName] = watcher
}

//查找服务
//负载均衡轮询策略
func (mng *ConsulMng) RoundSelect(serverName string) (*AddrInfo, error) {
	watcher := mng.watchers[serverName]
	if watcher != nil {
		return watcher.RoundSelect()
	}
	return nil, errors.New("ConsulMng RoundSelect serverName:" + serverName + " cannot find watcher!")
}

//固定id
func (mng *ConsulMng) GetServerInfo(serverName string, serverID string) (*AddrInfo, error) {
	watcher := mng.watchers[serverName]
	if watcher != nil {
		b, info, err := watcher.GetServerInfo(serverID)
		if !b && err == nil {
			//没有找到id 对应的服务信息，返回了一个随机的服务
			fmt.Println("ConsulMng GetServerInfo serverName:" + serverName + " get a rand server info!")
		}
		return info, err
	}
	return nil, errors.New("ConsulMng GetServerInfo serverName:" + serverName + " cannot find watcher!")
}

//--------------------
//监听信息
type ConsulWatcher struct {
	sync.RWMutex
	serverName string
	watchPlan  *watch.Plan
	//正常发现的地址 对于addrs读写锁
	addrsArr  []*AddrInfo
	addrsMap  map[string]*AddrInfo
	roundNext int
	rand      *rand.Rand
}

type AddrInfo struct {
	ID   string
	Addr string
}

func NewConsulWatcher(consulAddr string, serverName string) (*ConsulWatcher, error) {
	wp, err := watch.Parse(map[string]interface{}{
		"type":    "service",
		"service": serverName,
	})

	if err != nil {
		return nil, err
	}

	w := &ConsulWatcher{
		serverName: serverName,
		watchPlan:  wp,
		addrsArr:   []*AddrInfo{},
		addrsMap:   make(map[string]*AddrInfo),
		roundNext:  0,
		rand:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	wp.Handler = w.watchHandle
	go wp.Run(consulAddr)
	return w, nil
}

//轮询负载均衡
func (wth *ConsulWatcher) RoundSelect() (*AddrInfo, error) {
	wth.RLock()
	defer wth.RUnlock()
	lenA := len(wth.addrsArr)
	if lenA == 0 {
		return nil, errors.New("ConsulWatcher RoundSelect serverName:" + wth.serverName + " addrs is empty!")
	}
	addrInfo := &AddrInfo{}
	wth.roundNext = wth.roundNext % lenA
	addrInfo.ID = wth.addrsArr[wth.roundNext].ID
	addrInfo.Addr = wth.addrsArr[wth.roundNext].Addr
	wth.roundNext = (wth.roundNext + 1) % lenA
	return addrInfo, nil
}

//根据serverid获取server信息，地址数组未空返回error，未查找到返回false和一个随机地址，
func (wth *ConsulWatcher) GetServerInfo(serverID string) (bool, *AddrInfo, error) {
	wth.RLock()
	defer wth.RUnlock()
	lenA := len(wth.addrsArr)
	if lenA == 0 {
		return false, nil, errors.New("ConsulWatcher GetServerInfo serverName:" + wth.serverName + " addrs is empty!")
	}
	addrInfo := &AddrInfo{}

	//查找到
	isfind := wth.addrsMap[serverID]
	if isfind != nil {
		addrInfo.ID = isfind.ID
		addrInfo.Addr = isfind.Addr
		return true, addrInfo, nil
	}

	//未查找到随机返回
	rd := wth.rand.Intn(lenA)
	addrInfo.ID = wth.addrsArr[rd].ID
	addrInfo.Addr = wth.addrsArr[rd].Addr

	return false, addrInfo, nil
}

func (wth *ConsulWatcher) watchHandle(idx uint64, data interface{}) {
	entries, ok := data.([]*api.ServiceEntry)
	if !ok {
		return
	}

	//对addrs进行加锁，其他协成会访问
	wth.Lock()
	wth.addrsArr = []*AddrInfo{}
	wth.addrsMap = make(map[string]*AddrInfo)
	for _, entry := range entries {
		for _, check := range entry.Checks {
			if check.ServiceID == entry.Service.ID && check.Status == api.HealthPassing {
				addr := fmt.Sprintf("%s:%d", entry.Service.Address, entry.Service.Port)
				addrinfo := &AddrInfo{
					Addr: addr,
					ID:   entry.Service.ID,
				}
				wth.addrsArr = append(wth.addrsArr, addrinfo)
				wth.addrsMap[addrinfo.ID] = addrinfo
				break
			}
		}
	}
	wth.Unlock()
}
