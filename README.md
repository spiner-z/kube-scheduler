# Scheduler Plugin Demo

## Pre knowledge

### Golang

#### 接口

```go
// 定义接口
type Shape interface {
    Area() float64
    Perimeter() float64
}

// 定义一个结构体
type Circle struct {
    Radius float64
}

// Circle 实现 Shape 接口
func (c Circle) Area() float64 {
    return math.Pi * c.Radius * c.Radius
}
func (c Circle) Perimeter() float64 {
    return 2 * math.Pi * c.Radius
}

// 调用
func main() {
    c := Circle{Radius: 5}
    var s Shape = c // 接口变量可以存储实现了接口的类型
    fmt.Println("Area:", s.Area())
    fmt.Println("Perimeter:", s.Perimeter())
}

```

接口可以通过嵌套组合

```go
type Reader interface {
        Read() string
}
type Writer interface {
        Write(data string)
}

// ReadWriter 接口包含了 Reader 接口的 Read() 和 Writer 接口的 Write(data string)
type ReadWriter interface {
        Reader
        Writer
}

// 在 File struct 中实现
type File struct{}

func (f File) Read() string {
        return "Reading data"
}

func (f File) Write(data string) {
        fmt.Println("Writing data:", data)
}

func main() {
        var rw ReadWriter = File{}
        fmt.Println(rw.Read())
        rw.Write("Hello, Go!")
}
```

#### 继承

通过组合(composition)和接口(interface)来实现继承的功能

###### composition

```go
// 父结构体
type Animal struct {
    Name string
}

// 父结构体的方法
func (a *Animal) Speak() {
    fmt.Println(a.Name, "says hello!")
}

// 子结构体
type Dog struct {
    Animal // 嵌入 Animal 结构体
    Breed  string
}

func main() {
    dog := Dog{
        Animal: Animal{Name: "Buddy"}, // 初始化
        Breed:  "Golden Retriever",
    }

    dog.Speak() // 调用父结构体的方法
    fmt.Println("Breed:", dog.Breed)
}
```

##### interface

```go
// 定义接口
type Speaker interface {
    Speak()
}

// 父结构体
type Animal struct {
    Name string
}

// 实现接口方法
func (a *Animal) Speak() {
    fmt.Println(a.Name, "says hello!")
}

// 子结构体
type Dog struct {
    Animal
    Breed string
}

func main() {
    var speaker Speaker

    dog := Dog{
        Animal: Animal{Name: "Buddy"},
        Breed:  "Golden Retriever",
    }

    speaker = &dog
    speaker.Speak() // 通过接口调用方法
}
```

### kubernetes operator



## 组合默认调度插件

调整组合已有的默认插件，从而定义新的调度器。默认插件 `NodeResourcesFit` 有三种评分策略：`LeastAllocated`(默认)、`MostAllocated` 和 `RequestedToCapacityRatio`，这三种策略的目的分别是优先选择资源使用率最低的节点、优先选择资源使用率较高的节点从而最大化节点资源使用率、以及平衡节点的资源使用率。默认插件 `VolumeBinding` 绑定卷的默认超时时间是 600 秒。

以下例子自定义一个调度器：将 `NodeResourcesFit` 的评分策略配置为 `MostAllocated`，`VolumeBinding` 的超时时间配置为 60 秒。

**配置 `KubeSchedulerConfiguration`**

通过 `KubeSchedulerConfiguration` 对象自定义了一个调度器：

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
- schedulerName: my-custom-scheduler # 调度器名称
  plugins:
    score:
      enabled:
      - name: NodeResourcesFit
        weight: 1
  pluginConfig:
  - name: NodeResourcesFit
    args:
      scoringStrategy:
        type: MostAllocated
        resources:
        - name: cpu
          weight: 1
        - name: memory
          weight: 1
  - name: VolumeBinding
    args:
      bindTimeoutSeconds: 60
```

可以放在configmap里：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-scheduler-config
  namespace: kube-system
data:
  my-scheduler-config.yaml: |
    apiVersion: kubescheduler.config.k8s.io/v1
    kind: KubeSchedulerConfiguration
    profiles:
    - schedulerName: my-custom-scheduler # 调度器名称
      plugins:
        score:
          enabled:
          - name: NodeResourcesFit
            weight: 1
      pluginConfig:
      - name: NodeResourcesFit
        args:
          scoringStrategy:
            type: MostAllocated
            resources:
            - name: cpu
              weight: 1
            - name: memory
              weight: 1
      - name: VolumeBinding
        args:
          bindTimeoutSeconds: 60
```

**部署 Deployment 应用 kube-scheduler**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-custom-kube-scheduler
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      component: my-custom-kube-scheduler
  template:
    metadata:
      labels:
        component: my-custom-kube-scheduler
    spec:
      # serviceAccountName 注意需要配置权限
      containers:
      - command:
        - kube-scheduler
        - --leader-elect=false
        - --config=/etc/kubernetes/my-scheduler-config.yaml
        - -v=5
        image: registry.k8s.io/kube-scheduler:v1.31.2
        name: kube-scheduler
        volumeMounts:
        - name: my-scheduler-config
          mountPath: /etc/kubernetes/my-scheduler-config.yaml
          subPath: my-scheduler-config.yaml
      volumes:
      - name: my-scheduler-config
        configMap:
          name: my-scheduler-config
```

**部署 Pod 验证**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-default
spec:
  # schedulerName 默认就是使用 default-scheduler
  containers:
  - image: nginx
    name: nginx

---
apiVersion: v1
kind: Pod
metadata:
  name: nginx-custom
spec:
  schedulerName: my-custom-scheduler
  containers:
  - image: nginx
    name: nginx
```



## Sceduling framework

参考 [官方文档](https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/scheduling-framework/) 和更详细的 [设计文档](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/624-scheduling-framework/README.md)

一个 Pod 的调度流程包括 scheduling cycle 和 binding cycle 两个阶段

![image](https://github.com/kubernetes/enhancements/raw/master/keps/sig-scheduling/624-scheduling-framework/scheduling-framework-extensions.png)

### Extension points

一个 plugin 可以注册一个或多个 extension points。

#### Queue sort

用于对 scheduling queue 中的 pod 进行排序。提供一个 `less(pod1, pod2)` 函数

#### PreFilter

这里的插件可以对 Pod 进行预处理，或者条件检查，函数签名如下：

```go
// https://github.com/kubernetes/kubernetes/blob/v1.28.4/pkg/scheduler/framework/interface.go

type PreFilterPlugin interface {
    PreFilter(ctx , state *CycleState, p *v1.Pod) (*PreFilterResult, *Status)

    PreFilterExtensions() PreFilterExtensions
}
```

#### Filter

在这里过滤出不能运行 Pod 的节点。**任何一个插件** 返回失败，这个 node 就被排除了。

```go
// https://github.com/kubernetes/kubernetes/blob/v1.28.4/pkg/scheduler/framework/interface.go
type FilterPlugin interface {
    Plugin
    Filter(ctx , state *CycleState, pod *v1.Pod, nodeInfo *NodeInfo) *Status
}
```

#### PostFilter

只会在未找到合适节点的时候调用。

任意一个 PostFilter plugin 将某节点标记为可调度，剩下的 plugins 将不再被调度。一个比较典型的设计就是抢占（preemption）。

```go
// https://github.com/kubernetes/kubernetes/blob/v1.28.4/pkg/scheduler/framework/interface.go

type PostFilterPlugin interface {
    PostFilter(ctx , state *CycleState, pod *v1.Pod, filteredNodeStatusMap NodeToStatusMap) (*PostFilterResult, *Status)
}
```

#### PreScore

对于通过过滤的节点列表，更新内部状态或生成 logs/metrics。即生成一个可共享状态供 Score 插件使用。

#### Scoring

Scoring plugins 包含两个阶段：

第一个阶段 score：调度器会为每个过滤出来的节点调用每个 Scoring plugin，从而对节点进行排序。

第二个阶段 normalize scoring：这些插件用于在调度器计算 Node 排名之前修改分数。例如，假设一个 `BlinkingLightScorer` 插件基于具有的闪烁指示灯数量来对节点进行排名。

```go
func ScoreNode(_ *v1.pod, n *v1.Node) (int, error) {
    return getBlinkingLightCount(n)
}
```

然而，最大的闪烁灯个数值可能比 `NodeScoreMax` 小。要解决这个问题， `BlinkingLightScorer` 插件还应该注册该扩展点。

```go
func NormalizeScores(scores map[string]int) {
    highest := 0
    for _, score := range scores {
        highest = max(highest, score)
    }
    for node, score := range scores {
        scores[node] = score*NodeScoreMax/highest
    }
}
```

#### Reserve

Reserve 阶段发生在调度器实际将一个 Pod 绑定到其指定节点之前。 它的存在是为了防止在调度器等待绑定成功时发生竞争情况。 

#### Permit

Permit 插件在每个 Pod 调度周期的最后调用，用于防止或延迟 Pod 的绑定。

一旦所有 Permit 插件批准 Pod 后，该 Pod 将被发送以进行绑定。

如果任何 Permit 插件拒绝 Pod，则该 Pod 将被返回到调度队列。

如果一个 Permit 插件返回“等待”结果，则 Pod 将保持在一个内部的“等待中” 的 Pod 列表，同时该 Pod 的绑定周期启动时即直接阻塞直到得到批准。 如果超时发生，等待变成拒绝，并且 Pod 将返回调度队列。

### 插件 API

插件 API 分为两个步骤。首先，插件必须完成注册并配置，然后才能使用扩展点接口。 扩展点接口具有以下形式。

```go
type Plugin interface {
    Name() string
}

type QueueSortPlugin interface {
    Plugin
    Less(*v1.pod, *v1.pod) bool
}

type PreFilterPlugin interface {
    Plugin
    PreFilter(context.Context, *framework.CycleState, *v1.pod) error
}

// ...
```

### 调度器插件配置

在调度器配置中启用或禁用插件。参考官方文档 [调度器配置](https://kubernetes.io/zh-cn/docs/reference/scheduling/config/#scheduling-plugins)

可以通过编写配置文件，并将其路径传给 `kube-scheduler` 的命令行参数，定制 `kube-scheduler` 的行为。

调度模板（Profile）允许配置 kube-scheduler 中的不同调度阶段。每个阶段都暴露于某个扩展点中。插件通过实现一个或多个扩展点来提供调度行为。

通过运行 `kube-scheduler --config <filename>` 来设置调度模板， 使用 KubeSchedulerConfiguration v1 结构体：

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
clientConnection:
  kubeconfig: /etc/srv/kubernetes/kube-scheduler/kubeconfig
```

对每个扩展点，可以禁用默认插件或者启用自己的插件，例如：

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
  - plugins:
      score:
        disabled:
        - name: PodTopologySpread
        enabled:
        - name: MyCustomPluginA
          weight: 2
        - name: MyCustomPluginB
          weight: 1
```

****

下面默认启用的插件实现了一个或多个扩展点：

- `ImageLocality`：选择已经存在 Pod 运行所需容器镜像的节点。

  实现的扩展点：`score`。

- `TaintToleration`：实现了[污点和容忍](https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/taint-and-toleration/)。

  实现的扩展点：`filter`、`preScore`、`score`。

- `NodeName`：检查 Pod 指定的节点名称与当前节点是否匹配。

  实现的扩展点：`filter`。

- `NodeAffinity`：实现了[节点选择器](https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) 和[节点亲和性](https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity)。

  实现的扩展点：`filter`、`score`。

- `NodeUnschedulable`：过滤 `.spec.unschedulable` 值为 true 的节点。

  实现的扩展点：`filter`。

- `NodeResourcesFit`：检查节点是否拥有 Pod 请求的所有资源。 得分可以使用以下三种策略之一：`LeastAllocated`（默认）、`MostAllocated` 和 `RequestedToCapacityRatio`。

  实现的扩展点：`preFilter`、`filter`、`score`。

- `NodeResourcesBalancedAllocation`：调度 Pod 时，选择资源使用更为均衡的节点。

  实现的扩展点：`score`。

> 更多内容参考官方文档 [调度器配置](https://kubernetes.io/zh-cn/docs/reference/scheduling/config/#scheduling-plugins) 和源码

## 实践：Sticky pod

### stickyjob crd

定义 `stickyjob` 这个 crd：

```yaml
# stickyjob-crd.yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  # 名字必需与下面的 spec 字段匹配
  # 格式为 '<名称的复数形式>.<组名>'
  name: stickyjobs.example.com
spec:
  # 组名称，用于 REST API：/apis/<组>/<版本>
  group: example.com
  # 列举此 CustomResourceDefinition 所支持的版本
  versions:
  - name: v1
    # 每个版本都可以通过 served 标志来独立启用或禁止
    served: true
    # 其中一个且只有一个版本必需被标记为存储版本
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              template:
                type: object      # 对应 PodTemplateSpec
          status:
            type: object
            properties:
              lastNode:
                type: string      # 记录上次调度节点名
    subresources:
      status: {}
  # spec.scope 可以是 Namespaced 或 Cluster
  scope: Namespaced
  names:
    # 名称的复数形式，用于 URL：/apis/<组>/<版本>/<名称的复数形式>
    plural: stickyjobs
    # 名称的单数形式，作为命令行使用时和显示时的别名
    singular: stickyjob
    # kind 通常是单数形式的驼峰命名（CamelCased）形式。
    kind: StickyJob
```



## Reference

[K8s 调度框架设计与 scheduler plugins 开发部署示例（2024）](https://arthurchiao.art/blog/k8s-scheduling-plugins-zh/)

