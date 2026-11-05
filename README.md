# AI Native 课程与实践

打开 [课程首页](index.html) 查看课程和配套项目。下载仓库后，也可以在浏览器中直接打开根目录的 `index.html`。

## 课程

- [AI Native 产品设计实践](courses/ai-native-product-design-practice/index.html)
- [全栈 Agent 开发实践](courses/go-agent-engineering/index.html)
- [DSH 插件开发](courses/dsh-plugin/index.html)

## 示例项目

- [学业助手 eino-oa](examples/eino-oa/README.md)：Go、Eino 和 React 示例。
- [会议行动项评测台](examples/meeting-eval/README.md)：练习评测和比较 AI 输出。

运行环境、配置和启动步骤请查看各项目的 README。`assets/` 存放课程页面共用的样式、脚本和其他资源。

## 网页发布（GitHub Pages）

课程页面是静态 HTML、CSS 和 JavaScript，无需构建。根目录的 `.nojekyll` 让 GitHub Pages 直接发布现有文件。

在仓库 **Settings → Pages** 中，将 Source 设为 **Deploy from a branch**，选择 **main** 分支与 **/ (root)**，保存后自动部署。之后推送到 `main` 会自动更新网站。

默认访问地址：<https://vfan.github.io/ljb-ai-native/>。

Pages 托管课程网页；学业助手和会议评测台的 Go 服务仍按各自 README 在本地运行或另行部署。私有仓库需要支持 GitHub Pages 的账户套餐。
