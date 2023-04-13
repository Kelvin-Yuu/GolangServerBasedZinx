# GolangServerBasedZinx

> **说明**：本项目是基于Aceld大佬开发的Zinx服务器框架而成，按照教程“手把手”一步一步迭代搭建的服务器框架，仅供学习使用，无商业用途。
> ## Zinx源码地址
> ### Github
> Git: https://github.com/aceld/zinx
> ### 码云(Gitee)
> Git: https://gitee.com/Aceld/zinx
> ### 官网
> http://zinx.me
> ### 服务器框架开发搭建教程
> 【zinx-Golang轻量级TCP服务器框架(适合Go语言自学-深入浅出)】 
> https://www.bilibili.com/video/BV1wE411d7th

---
## 目前已经完成初步的框架搭建，命名为Zinx V1.0
### 功能简述

本框架是一个基于Golang的轻量级TCP并发服务器框架；

**支持多路由模式**，处理由用户绑定的自定义业务。多路由实现方式为，消息的MsgID与一个业务进行捆绑，存放在一个Map中；

**读写分离模型**，创建两个Goroutine与客户端进行数据交互，一个负责读业务，一个负责写业务。当服务端建立与客户端的套接字后，那么就会开启两个Goroutine分别处理读数据业务和写数据业务，读写数据之间的消息通过一个Channel传递；

**引入消息队列与多任务机制**，尽管golang在调度算法已经做的很极致，但大数量的Goroutine调度依然会带来一些不必要的环境切换成本。我们可以通过自行设定一个worker数量来限定处理业务的Goroutine数量，引入消息队列机制来缓冲worker工作的数据；

**链接管理**，为框架限定链接数，以保证后端的及时响应，如果超过一定量的客户端个数，会拒绝链接请求。


