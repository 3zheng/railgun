#railgun
一个简单的CS模式的游戏服务器框架，引入protobuf第三方库作为报文协议，mysql作为数据库，Go-mysql-driver作为驱动库。这三个东西是需要自己另外安装

##部署和编译
使用命令 go get -u github.com/3zheng/railgun 来download
如果不想使用go get -u github.com/3zheng/railgun命令可以去 http://git.oschina.net/poorbreast/railgun 这里开源中国上下载非import github.com/3zheng/railgun的工程源码

编译
GateApp,RouterApp,LoginApp是package main的exe工程，所以可以单独拿出来放到$GOPATH目录下，不会影响编译

##1.服务器架构
参考链接 http://blog.csdn.NET/easy_mind/article/details/53321919

##2.单个服务器APP结构
参考链接 http://blog.csdn.Net/easy_mind/article/details/53322216

##3.报文层级
参考链接 http://blog.csdn.net/easy_mind/article/details/53322280

##4.通过代码来简单说明
参考链接 http://blog.csdn.net/easy_mind/article/details/53322300

##5.目前存在的不足和后续可能的工作展开
参考链接 http://blog.csdn.net/easy_mind/article/details/53322687

##6.如何应用简述
每当根据业务需求新写一个业务App时，需要手写的源文件有
package main的main.go、PrivateMsg.go（这个如果没有需要新增私有报文就直接复制过来就行了）、XXXMsgFilter.go、XXXMainLogic.go、XXXDBLogic.go（如果需要操作数据库的话）

Package bs_proto的SetBaseInfo.go中的SetBaseKindAndSubId函数，要根据proto数据类型对其new Base并对Base.KindId和SubId赋值

##7.个人的吐槽
由于不知道怎么排版README，所以项目就在这里简述一下，详细文档说明移步到我的blog。个人水平有限，希望抛砖引玉吸引大神or练手的同学来将整个框架更加完善，共同为独立游戏开发者这个群体尽一点绵薄之力。希望感兴趣的同学or大神能和我一起完善开发，如果在征求您同意的前提下，我会将您的名字放入下面的贡献者名单中。我目前在369793160这个群里面，里面很多golang的高手，如果要共同学习进步可以加这个群。不过我不是群主也不是管理员，是否能加的进来看运气了，O(∩_∩)O哈哈~。如果真的用的人挺多的，那我届时会再建一个群，当前有问题可以发邮件到914509007@qq.com给我或者加我QQ，我有空都会看的

正常画风：如果项目对你有帮助，请点个☆，你的随手点赞是我继续改进的动力。
恶意卖萌画风：给人家点个☆嘛，点一下又不会怀孕(;¬_¬) 没粉丝就没动力，没时间。全部都是时辰的错

文档连接：http://blog.csdn.net/easy_mind/article/details/53260574

贡献者名单：
中二病也要当码畜	email:914509007@qq.com

PS:项目名源于《某科学的超电磁炮》，自曝死宅属性，233333。另外我要对我的死宅同胞们膜一句:当年花前月下的时候叫人家美琴酱（炮姐），如今新人胜旧人就叫人家上条夫人（or电磁炮）。你们呀就是有点好，出了新番比西方记者跑的都快，叫起“老婆”来一点都不含糊，实在是too young， too simple，sometimes naive。暗牧安谷瑞，不说了，我去丢我蕾姆去了。23333，逃 ε=ε=ε=(~￣▽￣)~