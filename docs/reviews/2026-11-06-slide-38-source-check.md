# 第 38 页：来源与示例口径核查

核查日期：2026-11-06。涉及页面：第 38 页监测过程，以及第 39 页使用的同一分类示例。

## 核查结论

三项检查有实际产品文档和通用评测方法支持。课件中的 10 个问题、每题 3 次、每轮 30 条以及青禾的回答片段是自设示例，不能称为 Rank Prompt 的固定采样流程、行业标准或真实监测数据。

| 页面内容 | 来源支持 | 适用范围 |
|---|---|---|
| 01：记录并比较问题、模型、语言、地区等条件 | Rank Prompt 的报告配置文档说明平台、地区、语言和问题的配置；官方产品说明介绍仅比较共同问题及对条件变化作标记。 | 支持检查可比条件。记录时间不代表两轮必须发生在同一时刻。 |
| 02：记录实际回答数及失败情况 | 官方将可见性定义为提到品牌的回答数占所收集回答数的比例；报告说明记录重试后仍失败的问题，以及报告实际收集的回答。 | 课件的“计划 30、2 次失败、有效 28”是假设演示。必须说明完成情况和分母，不能据此保证前后样本可比或没有缺失偏差。 |
| 03：保存回答原文，复查是否提及与是否推荐 | 官方 Prompts 页面可查看每个问题是否提到品牌及回答原文。 | 支持原文回查。课件的“提及/正向推荐”是本例简化标注规则，不作为该产品正式推荐指标定义。 |

## 已修正的表述

- 将采样导语改成“示例设定”，避免把 3 次、30 条理解为固定要求。
- 将公式标成“本例提及率”，说明这里采用的统计口径。
- 将“可考虑青禾”改成“推荐青禾”，第 39 页同步，避免将列为候选和明确推荐混为一谈。
- 按用户要求，保持底部总结和“想一想”删除。

“30 条足够可靠”没有得到本轮核查支持。重复次数应服务于测量目标；多次采样不自动证明内容修改导致了变化。

## 直接来源

- [Rank Prompt：Understanding your visibility score](https://help.rankprompt.com/en/articles/33-understanding-your-visibility-score)：按回答计数的可见性公式、别名和同名实体处理、未完成报告的处理。
- [Rank Prompt：What is a report?](https://help.rankprompt.com/en/articles/21-what-is-a-report)：每个问题与平台产生回答，报告状态、重试与仍然失败时的说明。
- [Rank Prompt：Report configuration options](https://help.rankprompt.com/en/articles/23-report-configuration-options)：平台、地区、语言及问题配置，回答对应的模型信息。
- [Rank Prompt：Prompts: named vs not named](https://help.rankprompt.com/en/articles/34-prompts-named-vs-not-named)：查看回答原文、提及状态、筛选条件与时间。
- [Rank Prompt：Official Information](https://rankprompt.com/llm-info/)：Monitoring and Reporting 中对共同问题的比较及条件变更标记。
- [NIST AI RMF Core](https://airc.nist.gov/airmf-resources/airmf/5-sec-core/)：MEASURE 2.1 要求记录测试集、指标和工具细节；框架支持评测记录与可追溯性，不规定本页的三项划分或采样数量。
- [Vercite：Mentioned vs recommended](https://vercite.io/research/mentioned-vs-recommended)：厂商自行发布的研究，明确区分提及、列为候选和推荐，并公开其分类定义；不将其结果视为所有产品的通用标准。
- [The Dice Roll Method，arXiv v2](https://arxiv.org/abs/2609.04047v2)：预印本讨论重复查询审计，未支持通用的固定重复次数；本页未据此给出样本量阈值。
