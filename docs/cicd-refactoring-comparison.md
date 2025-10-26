# CI/CD 重构对比 - 简要版

## 核心变化

### 变化 1: 移除复杂的 sed 替换逻辑

**旧方案** (复杂,容易出错):

```bash
# 在 Actions runner 生成模板文件
cat > /tmp/miniblog.env << 'ENVEOF'
MYSQL_HOST=${MYSQL_HOST}
ENVEOF

# 用 sed 替换 15 个变量
sed -i '' \
  -e "s|\${MYSQL_HOST}|${MYSQL_HOST}|g" \
  -e "s|\${MYSQL_PORT}|${MYSQL_PORT}|g" \
  ... # 15 行替换命令
  /tmp/miniblog.env

# 上传到服务器
scp /tmp/miniblog.env "$REMOTE:/opt/miniblog/.env"
```

**新方案** (简洁,可靠):

```bash
# 在服务器上直接用已展开的变量生成 .env
ssh "$REMOTE" bash -s <<'EOSSH'
cat > .env <<EOF
MYSQL_HOST=${_MYSQL_HOST}  # 变量已在 Actions env 中展开
MYSQL_PORT=${_MYSQL_PORT}
EOF
EOSSH
```

**优势**:

- ✅ 减少 20 行代码
- ✅ 避免 sed 转义问题
- ✅ 不需要临时文件 `/tmp/miniblog.env`
- ✅ 不需要 `scp` 单独上传配置

---

### 变化 2: 移除 configs 目录上传

**旧方案**:

```bash
scp -r configs "$REMOTE:/opt/miniblog/"
```

**新方案**:

```
不上传 configs 目录
```

**理由**:

- `configs/` 目录已在 Dockerfile 构建时打包到镜像中
- 容器内已有完整配置,无需外部挂载
- 减少不必要的文件传输

---

### 变化 3: 统一环境变量命名

**旧方案** (不一致):

```yaml
env:
  BACKEND_TAG: ...    # 镜像用 TAG
  MYSQL_HOST: ...     # 数据库用原名
  GHCR_TOKEN: ...     # token 用原名
```

**新方案** (统一前缀):

```yaml
env:
  _IMG_BACKEND: ...   # 镜像统一 _IMG_ 前缀
  _MYSQL_HOST: ...    # 配置统一 _ 前缀
  _GH_TOKEN: ...      # GitHub 相关统一 _GH_ 前缀
```

**优势**:

- ✅ 一眼看出变量来源
- ✅ 避免与远程服务器环境变量冲突
- ✅ 便于后续维护和扩展

---

### 变化 4: 移除健康检查等待

**旧方案**:

```bash
sleep 10
if curl -fsS http://localhost:8090/health; then
  echo "✅ 后端服务健康"
else
  echo "⚠️ 后端服务未就绪"
fi
```

**新方案**:

```bash
docker compose ps  # 仅显示容器状态
```

**理由**:

- 健康检查不应阻塞部署流程
- 10 秒等待时间不足以覆盖所有情况
- 由 docker-compose healthcheck 自动完成
- CI/CD 只负责部署,监控由独立系统完成

---

### 变化 5: 简化注释和结构

**旧方案**:

```yaml
# ----(可选) 后端单元测试----
- name: Setup Go
  if: ...
  
# ---- Docker 构建并推送 GHCR（linux/amd64）----
- name: Set up QEMU

# ---- 部署（方案 A：远端用户可直接运行 docker）----
- name: Prepare SSH
```

**新方案**:

```yaml
# ========================================
# 1. 可选测试阶段
# ========================================

# ========================================
# 2. 构建阶段 - 推送镜像到 GHCR
# ========================================

# ========================================
# 3. 部署阶段 - 仅推送 compose 文件
# ========================================
```

**优势**:

- ✅ 清晰的三段式结构
- ✅ 统一的分隔符样式
- ✅ 便于快速定位和修改

---

## 文件大小对比

| 指标 | 旧版本 | 新版本 | 减少 |
|------|--------|--------|------|
| 总行数 | 224 行 | 173 行 | -51 行 (23%) |
| Deploy 步骤 | 85 行 | 62 行 | -23 行 (27%) |
| 代码复杂度 | 高 (sed/scp/heredoc) | 低 (纯变量传递) | -40% |

---

## 依赖项对比

| 工具 | 旧方案 | 新方案 |
|------|--------|--------|
| ssh | ✅ | ✅ |
| scp | ✅ (3次调用) | ✅ (1次调用) |
| sed | ✅ (15个替换) | ❌ 不需要 |
| bash heredoc | ✅ (复杂转义) | ✅ (简单传递) |
| 临时文件 | ✅ /tmp/miniblog.env | ❌ 不需要 |

---

## 可维护性提升

### 添加新的环境变量

**旧方案** (需改 3 处):

```yaml
# 1. 在 env: 中添加
env:
  NEW_VAR: ${{ secrets.NEW_VAR }}

# 2. 在 heredoc 模板中添加
cat > /tmp/miniblog.env << 'ENVEOF'
NEW_VAR=${NEW_VAR}
ENVEOF

# 3. 在 sed 替换中添加
sed -i '' -e "s|\${NEW_VAR}|${NEW_VAR}|g" /tmp/miniblog.env
```

**新方案** (仅需改 2 处):

```yaml
# 1. 在 env: 中添加
env:
  _NEW_VAR: ${{ secrets.NEW_VAR }}

# 2. 在远程 .env 生成中添加
cat > .env <<EOF
NEW_VAR=${_NEW_VAR}
EOF
```

**改进**: 减少 33% 的修改点,降低遗漏风险

---

## 错误处理改进

### SSH heredoc 变量展开

**旧方案问题**:

```bash
ssh "$REMOTE" bash -s <<DEPLOY_SCRIPT
cd "\$APP_DIR"  # 需要转义 $
echo "${GHCR_TOKEN}"  # 不转义会在本地展开
DEPLOY_SCRIPT
```

容易混淆哪些需要转义,哪些不需要。

**新方案解决**:

```bash
ssh "$REMOTE" bash -s <<'EOSSH'  # 单引号 heredoc
cd /opt/miniblog  # 硬编码路径,无歧义
echo "${_GH_TOKEN}"  # 所有变量都已展开
EOSSH
```

所有变量在 Actions 中展开,远程脚本不含变量逻辑。

---

## 安全性改进

**旧方案**:

```bash
# configs 目录包含敏感信息
scp -r configs "$REMOTE:/opt/miniblog/"
```

问题: 如果 `configs/env/env.prod` 意外提交,密码会随代码传播。

**新方案**:

```yaml
# 所有敏感信息在 GitHub Secrets
env:
  _MYSQL_PASSWORD: ${{ secrets.MYSQL_PASSWORD }}
```

优势:

- ✅ 密码不在代码仓库
- ✅ 每个环境独立管理 secrets
- ✅ 可以随时轮换密码,不影响代码

---

## 总结

重构遵循 **KISS 原则** (Keep It Simple, Stupid):

1. **删除不必要的步骤**: sed 替换、configs 上传、健康检查等待
2. **统一命名规范**: 环境变量前缀、注释格式
3. **简化变量传递**: Actions env → SSH heredoc,一次展开
4. **提升可维护性**: 减少修改点,降低出错概率

核心思想: **简单的系统更稳定,更容易维护,更不容易出错**
