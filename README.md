go-crawling-bazaar是一个基于golang实现的去中心化爬虫平台.

## 去中心化爬虫系统
在大数据时代，对于大规模、分布式爬虫的需求日益增加，这要求爬虫设计者要么投入大量设备进行爬取工作，要么购买昂贵的云主机进行爬取，或者将爬取工作外包给拥有大量设备的爬虫服务提供商，但是这些方法使用需要投入大量的金钱或者资源，有没有一种更加廉价的网页数据获取方式呢？受到P2P和去中心化的区块链的启发，我们设计了一个去中心化的爬虫网络。这个网络是由大量的第三方的爬虫节点构成，这些爬虫节点收到来自爬虫客户端的请求后，就会根据一定的算法随机选出少量节点对目标URL进行爬取，爬取结果经过验证后，将存入区块链中，爬虫客户端支付一小笔爬取费用给爬取节点就可以得到页面内容。

## 设计思路
和由大量可信节点构成的中心化爬虫系统（比如scrapy-redis）不同，在我们设计的系统中，任何人都可以搭建爬虫节点并加入爬虫网络，因此，我们面对的是大量不可信节点，如何让若干不可信节点去可信地完成爬取任务，是需要面对并加以解决的问题。
为此，在2019~2020年我们开发了go-decentralized-frontera，在这个项目中，我们使用区块链的智能合约来记录爬取过程中的请求、回应，并记录各种中间状态，这使得爬虫系统费用高，性能差，性价比很低，没法在实际环境中使用.

为了改进这点，我们进行了重新设计，并开发了go-crawling-bazaar这个项目，在这个项目中区块链仅仅用来保存爬取结果，爬取过程则由链上转移到链下，对于一个特定的URL仅由的少量节点负责进行爬取。在这里，我们遇到的挑战是如何确保少量不可信节点可信地完成爬取工作？
我们使用的最主要的验证机制的原理是，如果若干随机挑选的爬虫节点的爬取结果是一致的（或者仅仅有微小的差别），则认为该爬取结果是可靠的。图1给出了Client和系统交互的过程，Client随机地（或者根据一定的指标）找到愿意替他进行爬取工作的Miner节点，并向它提出一个爬取任务（bidding），随后Miner节点将负责控制改爬取任务，并不断广播URL爬取请求（Request），接收到Request的Crawler节点将检查自己是否是本次请求的合格的爬取者，只有那些合格的Crawler节点会进行实际的爬取工作，并将爬取结果保存在DHT存储网络中，并将Result（页面内容的Cid）返回给Miner节点。Miner节点将检查来自若干Crawler返回的结果，如果这些结果能够互相印证，则将URL、Cid以及实际进行爬取工作的Crawler的ID记录到区块链上，当整个爬取任务完成之后，Miner节点将同步、或者异步地通知Client取回爬取结果（Bid）。
 
![Bidding bid process](docs/bidding-bid.jpg)

为了防止群鸽效应，既一次请求导致对同一页面的多次访问，我们一般限制对同一个Request的进行处理的合格节点数为k，k一般小于等于3， 为了实现这个抽签机制，通常采用公开的信息（比如URL等）作为输入，通过公开的随机函数计算一个0~1之间的中签概率p，如果p < k/n 则当前节点抽中。由于所有信息均为公开，攻击者有一定的时间窗口能够预测出抽签结果对中签节点实施攻击，或者和中签者进行合谋骗取Client的token。
为了解决这个问题，我们引入了VRF（verifiable random function），由于使用区块链高度+节点私钥作为Input，VRF的结果无法被预测，其它节点只有通过网络接收到随机结果后才能对其合法性进行验证，即攻击者在得知抽签结果时，中签节点已经完成爬取工作了，这样，就大大提高了攻击者实现合谋攻击的难度。

## 两层区块链网络
使用区块链完成这样的一个大型爬虫系统，最大的挑战就是平衡吞吐率和扩展性之间的矛盾，为此，我们借鉴了CovenantSQL，引入了二层网络，主网（Presbyteria Network）和负责各个URL Domain的爬虫网络（Domain 's Crawler Network），参考图2，其中主网的作用就是用来维护各个域爬虫网络的元信息，对于每一个URL Domain都使用一个单独的爬虫网络，其最大的好处是不同爬取域可以并行工作，这显著地提高了爬虫系统的吞吐量。
 
![2 layers blockchain](docs/two-layers-blockchain.jpg)

主网是由主网矿工（Presbyterian Network Miner，一下简称PB Miner）来维护的，于此相对，每个爬虫网络包括两种类型的节点：爬虫网络矿工（Crawling Network Miner，以下简称Crawling Miner、或Miner）和爬虫节点（Crawler），两种矿工节点（包括PB Miner和Crawling Miner）都靠维护网络来得到报酬，而Crawler节点则通过为用户提供爬取服务而得到报酬。

## 共识算法
- 主网共识算法
和CovenantSQL类似，Presbyterian Network使用DPoS (委任权益证明) 作为共识协议 。

- 爬虫网络共识算法
（CovenantSQL存储网络使用了两种共识算法，一种是满足最终一致性的DPos共识，另外一种是为了提供强一致性而采用的BFT-Raft (拜占庭容错算法)bft-raft 共识算法），因为爬虫系统并不是一个事务系统，我们只需要保证最终一致性即可，为此，我们使用了基于爬取工作量的PoS共识算法，该算法的核心目标是使Miner出块的权益与自己对页面爬取的贡献成正比，也就是说，Miner在上一个epoch中有效爬取页面数在整个网络的爬取页面总数中所占的比例，就是此Miner能够出块的概率。

可能，一个更好的选择是Filecoin中的EC算法，因为EC满足下面几个特征：
公平性：算力大的矿工命中的机会大， 机会与算力成正比;
不可预知性：不能预测获胜节点（实际上是利用hash算法的不可逆来实现的）
可验证性：这个就涉及到VRF（可验证随机函数）的问题了。
可行性：预期共识仅仅需要每个矿工在本地计算就可以了，非常简单，实现起来很容易。
但是，EC算法存在一个比较大的问题，就是说每个节点命中的可能性是独立的。即每一轮中，有可能一个节点也选不出来；也有可能有好几个节点都命中，这样就会有好几个节点同时出块了。这就不是很理想了，因为有些轮没有人出块，交易（消息）的过程就会延长；如果有些轮有多个人出块，这些块中的消息信息基本上都是重复的，占用区块空间，浪费，这使得Filecoin的EC共识相关的代码变得非常复杂。
（所以呀，更好的算法当然是最好每轮都能出块，而且每一轮都只有一个矿工上靶（具有出块资格）。可以想象，这样的算法当然需要整个网络进行协调，进行真正的选举，那这样是不是复杂度提高了，是不是可行呢？目前有好些算法被提出来，可以查询CoA， Snow White， AlgoRand等。但是，有些过于复杂，有些不适合Filecoin。）
我们是否可以提出一个新的想法？

## 状态镜像
CQL 的完整数据是存在 SQLChain 的 Miner 上的，这部分数据相关的 SQL 历史是完整保存在 Miner 上的。相对于传统数据库的 CRUD （Create、Read、Update、Delete），CQL 支持的是 CRAP （Create、Read、Append、Privatize）。
Append vs Update：传统数据库对数据进行更改（Update）后是没有历史记录存在的，换句话说数据是可被篡改的。CQL 支持的是对数据进行追加（Append），其结果是数据的历史记录是得以保全。
Privatize vs Delete：传统数据库对数据进行删除（Delete）也属于对数据不可追溯、不可逆的篡改。CQL 支持的是对数据进行私有化（Privatize），也就是把数据库的权限转给一个不可能的公钥。这样就可以实现对子链数据的实质性删除。针对单条数据的链上所有痕迹抹除目前仅在企业版提供支持。

## 系统构成
为了让大量非scrapy爬虫将无法也能接入go-crawling-bazaar，我们将每个爬虫看成两个子系统构成的，一个是爬虫子系统，一个是frontier子系统，前者负责爬取工作，后者负责获取来自其他爬虫节点的URL请求、并将自己获得的URL目标传递给其他节点，爬虫子系统将通过一个API和frontier子系统进行沟通，这样，即使是用其他语言、其他框架实现的爬虫，经过简单的改造后，也可以充分利用我们的平台，实现去中心化的爬取工作。参考下面的图3
 
![system architecture](docs/go-crawling-bazaar-network.jpg)

因此，在我们的开源系统中，也包含了两个子项目，一个是go-crawling-bazaar, 一个是scrapy-crawling-bazaar，前者负责构成Crawling bazaar网络，后者则包含了在scrapy框架下调用go-crawling-bazaar的接口程序库以及一些基于scrapy的示范代码，用户可以模仿这些示范代码，将自己的单机爬虫改造成使用go-crawling-bazaar的去中心化爬虫。
7，支付系统


## 安装

## 执行
参考 *.sh

