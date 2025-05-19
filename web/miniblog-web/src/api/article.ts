import type { Article } from '../types/article'

// mock 数据
const mockArticles: Article[] = [
  {
    id: 1,
    sectionCode: 'ai_history',
    title: 'AI 的起源：达特茅斯会议',
    summary: '达特茅斯会议是人工智能领域的里程碑事件，它标志着人工智能作为一门独立学科的诞生。',
    content: '达特茅斯会议是人工智能领域的里程碑事件，它标志着人工智能作为一门独立学科的诞生。',
    author: 'clack',
    createdAt: '2025-05-01',
    updatedAt: '2025-05-01',
    tags: ['ai', '达特茅斯会议']
  },
  {
    id: 2,
    sectionCode: 'turing',
    title: 'AI 的定义：理解图灵',
    summary: '图灵测试是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '图灵测试是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['ai', '图灵测试']
  },
  {
    id: 3,
    sectionCode: 'prompt',
    title: '信息论与AI',
    summary: '信息论是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '信息论是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-03',
    updatedAt: '2025-05-03',
    tags: ['ai', '信息论']
  },
  {
    id: 4,
    sectionCode: 'golang_basic',
    title: 'AI 的定义：理解图灵',
    summary: '图灵测试是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '图灵测试是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['ai', '图灵测试']
  },
  {
    id: 5,
    sectionCode: 'concurrency',
    title: 'Go 语言的并发模型',
    summary: 'Go 语言的并发模型是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: 'Go 语言的并发模型是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['golang', '并发模型']
  },
  {
    id: 6,
    sectionCode: 'golang_basic',
    title: 'Go 中的值类型与引用类型',
    summary: 'Go 中的值类型与引用类型是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: 'Go 中的值类型与引用类型是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['golang', '值类型', '引用类型']
  },
  {
    id: 7,
    sectionCode: 'design_pattern',
    title: '架构设计',
    summary: '架构设计是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '架构设计是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['架构设计']
  },
  {
    id: 8,
    sectionCode: 'architecture',
    title: 'DDD 领域驱动设计',
    summary: 'DDD 领域驱动设计是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: 'DDD 领域驱动设计是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['架构设计', 'DDD']
  },
  {
    id: 9,
    sectionCode: 'architecture',
    title: '六边形架构',
    summary: '六边形架构是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '六边形架构是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['架构设计', '六边形架构']
  },
  {
    id: 10,
    sectionCode: 'architecture',
    title: '设计原则：单一职责原则',
    summary: '单一职责原则是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '单一职责原则是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['编程', '单一职责原则']
  },
  {
    id: 11,
    sectionCode: 'architecture',
    title: '设计原则：开闭原则',
    summary: '开闭原则是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '开闭原则是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['编程', '开闭原则']
  },
  {
    id: 12,
    sectionCode: 'architecture',
    title: '设计原则：里氏替换原则',
    summary: '里氏替换原则是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '里氏替换原则是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['编程', '里氏替换原则']
  },
  {
    id: 13,
    sectionCode: 'golang_basic',
    title: '分层价格',
    summary: '',
    content: `
## 定义：
分层架构是以分离关注点 MEMO 为原则，将系统按照功能和职责划进行多层划分，并分别进行实现的一种设计理念。
分层架构通过系统层次的划分和独立实现，极大的提高了每一层内部的内聚性，降低了层与层之间的耦合性，也是“高内聚、低耦合“设计原则的典型体现。
分层架构也是在软件开发领域对“分而治之“思想的运用，即将一个复杂问题拆分为多个小问题，分别解决后再整合到一起，从而解决整个大问题。

## 发展历程：
- 单体结构阶段（软件开发的早期）
  - 表现：没有明确的职责划分，几乎所有的用户操作、业务逻辑、数据存储代码都庞杂的堆砌在一起。
  - 问题：更改用户操作时可能影响业务逻辑、更改业务逻辑时又可能导致数据存储出问题，导致系统维护异常困难。
- MVC 分层架构
  - Martin Fowler 在《企业应用架构模式》中大力推广 MVC 分层架构，提倡将用户界面层（View）、业务逻辑层（Model/Controller）和数据存储（Data Access Layer）层相分离，有效解决单体结构的高耦合问题。
- 分离领域
  - Eric Evans 在领域驱动设计中进一步提出了 分离领域 的设计思想，强调领域层（Domain Layer）作为系统的核心，专注于表达业务逻辑和领域知识，并严格独立于基础设施层、应用服务层，以更好的应对复杂业务场景的建模。

## 优点&作用：
- 分层架构的设计，使得层级内部更有内聚性，并只依赖下层。
- 不同的层级是为了描述不同的问题，且以不同的速度发展，所以层级之间具有低耦合性，使得层级更清楚，更容易维护。

## 抽象化实现：
分层架构示意图：
- 分层架构将代码分成若干层，每层负责不同的关注点；
- 箭头标识依赖关系，外层只能依赖内层，内层不能依赖外层；
  - 这表明软件架构中的一个重要原则：代码中不稳定的部分，应该依赖稳定的部分；
  - 所以分层架构中越是内层越稳定，越是外层越容易变化。
![分层架构示意图](https://cdn.jsdelivr.net/gh/clack-cn/cdn/images/2025/05/19/1.png)
![分层架构示意图](https://cdn.jsdelivr.net/gh/clack-cn/cdn/images/2025/05/19/2.png)

## MVC 分层架构实现
MVC 分层架构是在软件开发中，以分离关注点为核心设计原则，通过划分出业务逻辑（Model）、用户界面（View）、用户交互控制（Controller）三个核心职责，来降低系统耦合度，提高系统可维护性、可扩展性的架构设计。

在企业级应用开发中，通常基于经典的 MVC 架构，还会独立出数据访问层（Data Access Layer / DAO），以便于更清楚的隔离业务逻辑和数据访问的依赖关系。

## 发展历程：
- 经典 MVC 架构
  - 提出者：Trygve Reenskaug 于 1979 年提出
  - 应用领域：最初用于桌面应用的用户界面设计
  - 具体描述：
    - Model（模型层）
      - 负责管理和维护应用程序的业务逻辑与业务数据。
    - View（视图层）
      - 负责用户界面的显示，接收用户输入，向用户呈现数据。
    - Controller（控制器层）
      - 接受用户输入事件，调用 Model 层处理业务逻辑，并更新视图，起到 Model 与 View 间的协调和控制作用。
  - 特点： 经典 MVC 的 Model 层通常涵盖了业务逻辑和数据访问逻辑，没有单独的数据访问层概念。
- Martin Fowler 提出的企业级三层架构（对经典 MVC 的扩展）
  - 提出者：Martin Fowler 于 2002 年在《企业应用架构模式》中提出
  - 应用领域：企业级应用架构设计
  - 具体描述：
    - 表现层（Presentation Layer）
      - 专注于用户界面的显示和用户交互处理，通常包含视图（View）和控制器（Controller）的部分职责。
    - 领域层（Domain Layer）
      - 专注于表达和实现核心业务逻辑和业务规则，与经典 MVC 中的 Model 层职责相似，但明确强调不包含数据访问逻辑。
    - 数据源层（Data Source Layer）
      - 专注于处理数据持久化（如数据库、缓存）等技术细节，通常以数据访问对象（DAO）的方式实现。
  - 特点：Fowler 明确提出领域层和数据访问层的分离，以适应企业级应用架构的更高维护性和扩展性需求。

## 实践中企业开发的分层架构（综合经典 MVC 与 Fowler 架构）
在实际的企业软件开发中，将经典 MVC 和 Fowler 三层架构结合，形成更清晰的五层架构模式：
| 层次 | 名称 | 职责描述 |
| ------- | ------- | ------- |
| 表现层 | View | 只负责界面的显示逻辑和用户输入 |
| 表现层 | Controller | 负责接收并处理用户请求，协调调用业务服务 |
| 服务层 | Service | 业务逻辑的聚合与协调，调用领域模型实现业务 |
| 领域模型层 | Domain Model | 表达纯粹的业务规则与领域知识，不涉及技术细节 |
| 数据访问层 | DAO | 处理数据持久化、数据库交互与外部资源访问 |

## DDD 分离领域 & 六边形架构
### DDD 分离领域
在领域驱动设计（DDD）中，分离领域 的设计思想强调将 领域层 作为系统的核心，专注于表达业务逻辑和领域知识。领域层应严格独立于基础设施层和应用服务层，目的是为了更好地应对复杂业务场景的建模，并保持业务逻辑的纯粹性与可重用性。

### 六边形架构（Hexagonal Architecture）
六边形架构（又称为 Ports and Adapters 架构）旨在创建一个与用户界面和数据库解耦的应用系统。通过这种方式，应用程序可以在没有依赖数据库或特定用户界面的情况下运行，从而便于进行自动化回归测试、脱离数据库的开发，并且能轻松地与其他系统进行连接和交互。
六边形架构强调将应用的核心与外部世界进行隔离，通过**端口（Ports）与适配器（Adapters）**来与外部环境（如数据库、用户界面、外部系统等）进行交互。

### 分离领域 & 六边形架构的层级架构
以下是分离领域与六边形架构的层级架构描述：
1. 领域层（Domain Layer）
  - 元素：领域对象、领域服务
  - 定义：领域层是系统的核心，专注于表达业务逻辑和领域知识。它不依赖于外部基础设施，体现业务规则和领域模型。
  - 职责：领域层实现的是“纯业务逻辑”，它独立于技术实现，保证了系统的灵活性和可维护性。
2. 应用层（Application Layer）
  - 元素：应用服务
  - 定义：应用层不包含业务逻辑，而是对领域层的业务逻辑进行封装、调用和编排。
  - 作用：
    - 接受客户端请求，并协调领域层进行业务处理。
    - 将领域层的处理结果封装为数据传输对象（DTO），并对外输出。
    - 负责处理事务、日志、权限等横切关注点。
3. 适配器层（Adapter Layer）
  - 元素：主动适配器（由外向内）、被动适配器（由内向外）
  - 定义：适配器层处理系统与外部的交互技术，分为两类：
    - 主动适配器：由外部系统触发，访问内部模块。例如：Restful API、RPC、Web 页面等。
    - 被动适配器：由系统内部发起，访问外部资源。例如：数据库持久化层、云文件访问层等。
  - 作用：
    - 分离关注点：通过适配器层将系统核心与外界隔离，保证内部模块的稳定性。
    - 扩展性：适配器层能够根据需要扩展多种不同的适配器，从而增加系统的灵活性。
4. 通用服务层（Common Layer）
  - 元素：工具类（Util）、框架性代码、公共库等
  - 定义：通用服务层包含系统中用于支撑其他层的基础类和工具类代码，提供系统所需的通用功能和服务。
  - 作用：
    - 为系统提供公共支持，简化其他层次的实现。
    - 例如：日志、缓存、配置管理、加密解密等基础功能。

## 架构示意图：
![分层架构示意图](https://cdn.jsdelivr.net/gh/clack-cn/cdn/images/2025/05/19/1.png)
![分层架构示意图](https://cdn.jsdelivr.net/gh/clack-cn/cdn/images/2025/05/19/2.png)
    `,
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['编程', '依赖倒置原则']
  }
]

// fetchArticlesBySectionCode 获取分类文章列表
export async function fetchArticlesBySectionCode(sectionCode: string): Promise<Article[]> {
  return new Promise(resolve => setTimeout(() => {
    console.log(`in fetchArticlesBySectionCode, sectionCode: ${sectionCode}`)

    // 查找 articles 数据
    var fetchedArticles = mockArticles.filter(a => a.sectionCode === sectionCode)

    resolve(fetchedArticles)
  }, 300))
}

// fetchArticleById 获取文章详情
export async function fetchArticleById(id: number): Promise<Article | undefined> {
  return new Promise(resolve => setTimeout(async () => {
    console.log(`in fetchArticleById, id: ${id}`)

    resolve(mockArticles.find(a => a.id === id))
  }, 300))
}