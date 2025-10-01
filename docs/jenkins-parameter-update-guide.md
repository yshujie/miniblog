# Jenkins 参数更新说明

## 🔍 问题现象

修改了 Jenkinsfile 中 `SKIP_DB_SEED` 的默认值从 `true` 改为 `false`，但是构建日志显示：

```
Stage "DB Seed" skipped due to when conditional
```

DB Seed 阶段仍然被跳过了。

## 📋 原因分析

### Jenkins 参数更新机制

Jenkins 的**参数化构建**有一个特点：

1. **第一次构建**：使用 Jenkinsfile 中定义的**旧参数值**
2. **Jenkins 解析 Jenkinsfile**：更新参数定义到 Jenkins 数据库
3. **第二次及以后构建**：使用**新的参数默认值**

### 为什么第一次构建没有生效

当前构建使用的参数值：

- **构建时的参数**：`SKIP_DB_SEED = true`（来自之前的参数定义）
- **代码中的逻辑**：`env.RUN_DB_SEED = shouldSkip(params.SKIP_DB_SEED, env.SKIP_DB_SEED) ? 'false' : 'true'`
- **结果**：`params.SKIP_DB_SEED = true` → `shouldSkip() = true` → `RUN_DB_SEED = 'false'` → 阶段被跳过

## ✅ 解决方案

### 方案 1：等待下次自动构建（推荐）

下次 Git push 触发的构建将自动使用新的参数默认值 `SKIP_DB_SEED = false`。

**操作步骤**：

1. 等待 Jenkins 自动触发新构建（通过 GitHub webhook）
2. 新构建将会执行 DB Seed 阶段
3. 数据加载完成

### 方案 2：手动触发新构建

在 Jenkins UI 中手动触发一次新的构建。

**操作步骤**：

1. 进入 Jenkins → miniblog 项目
2. 点击 "Build with Parameters"
3. 查看参数，现在 `SKIP_DB_SEED` 应该默认为 `false`（未勾选）
4. 点击 "Build" 开始构建
5. 这次构建将会执行 DB Seed

### 方案 3：在当前构建中手动修改参数

如果想立即执行，可以在 Jenkins UI 中重新触发构建并手动取消勾选 `SKIP_DB_SEED`。

**操作步骤**：

1. 进入 Jenkins → miniblog 项目
2. 点击 "Build with Parameters"
3. **取消勾选** `Skip loading seed data...`
4. 点击 "Build"
5. 这次构建将会执行 DB Seed

## 📊 验证方法

### 1. 查看构建日志

在 Setup 阶段，应该能看到：

```
Run db seed: true (SKIP_DB_SEED param: false)
```

而不是之前的隐式跳过。

### 2. 查看 DB Seed 阶段

如果参数正确，应该能看到：

```
[Pipeline] stage
[Pipeline] { (DB Seed)
[Pipeline] dir
Running in /var/jenkins_home/workspace/miniblog
[Pipeline] {
[Pipeline] sh
+ make db-seed
[load-seed-data] Loading seed data into database: miniblog
[load-seed-data] Using DB_HOST=mysql, DB_PORT=3306, DB_USER=miniblog
-> Using docker exec to load data
Loading user.sql...
✓ user.sql loaded successfully
Loading module.sql...
✓ module.sql loaded successfully
...
```

### 3. 验证数据库

构建完成后，连接数据库检查：

```bash
docker exec -it mysql mysql -uminiblog -p miniblog

SELECT COUNT(*) FROM user;      -- 应该有用户数据
SELECT COUNT(*) FROM module;    -- 应该有 6 个模块
SELECT COUNT(*) FROM article;   -- 应该有大量文章
```

## 📝 Jenkins 参数更新的一般规律

### 何时生效

| 场景 | 参数值来源 | 是否使用新默认值 |
|------|----------|---------------|
| 修改 Jenkinsfile 后的**第一次构建** | 旧的参数定义 | ❌ 否 |
| 修改 Jenkinsfile 后的**第二次构建** | 新的参数定义 | ✅ 是 |
| 手动触发"Build with Parameters" | Jenkins UI 显示的当前值 | ✅ 是 |

### 最佳实践

1. **修改参数默认值后**：
   - 知道第一次构建使用旧值
   - 准备好等待第二次构建或手动触发

2. **紧急情况**：
   - 使用 Jenkins UI 的 "Build with Parameters"
   - 手动调整参数值
   - 不依赖默认值

3. **生产环境**：
   - 重要操作（如数据加载）应该默认关闭（`true` = skip）
   - 需要时手动开启
   - 避免意外执行

## 🎯 当前状态总结

### 已完成

- ✅ Jenkinsfile 参数默认值已修改：`SKIP_DB_SEED: false`
- ✅ 添加了日志输出，便于调试
- ✅ 代码已推送到 GitHub

### 下一步

- ⏳ 等待下次构建自动触发（GitHub push）
- ⏳ 或者手动在 Jenkins UI 中触发新构建
- ⏳ 验证 DB Seed 执行并加载数据
- ⏳ **重要**：数据加载成功后，再次修改 `SKIP_DB_SEED` 默认值为 `true`

### 预期时间线

1. **现在**：第一次构建完成，DB Seed 被跳过（使用旧参数）
2. **下次 push 或手动触发**：第二次构建，DB Seed 将执行
3. **数据加载完成后**：修改参数默认值回到 `true`，避免重复加载

## 📞 需要帮助？

如果下次构建 DB Seed 仍然被跳过：

1. 检查 Setup 阶段的日志：`Run db seed: ?`
2. 检查参数值：`SKIP_DB_SEED param: ?`
3. 如果仍然是 `true`，尝试手动在 Jenkins UI 中触发构建
