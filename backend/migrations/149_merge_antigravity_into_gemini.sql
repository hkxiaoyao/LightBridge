-- 将 Antigravity 合并入 Gemini 平台。
--
-- 背景：历史上 Antigravity 是 accounts.platform 的一个独立取值。合并后，
-- Antigravity 账号统一以 platform='gemini' + sub_platform='antigravity' 存储；
-- sub_platform 作为同一平台下的账号变体判别符，与 type（oauth/apikey/upstream）正交
-- ——这正是无法直接复用 type 字段区分 Antigravity 的原因。
--
-- 兼容性：'antigravity' 仍作为“平台别名”保留，分组 platform、强制平台中间件、
-- 配额归因、历史 usage_logs 等均不受本迁移影响；本迁移仅改写 accounts 表。
--
-- 幂等：可重复执行——列存在则跳过新增；二次执行时已无 platform='antigravity' 行可改写。
--
-- 运维提示：本迁移在数据库层直接改写 platform，调度快照（Redis）可能短暂滞后，
-- 部署后建议触发一次调度快照重建或等待其自然刷新。

-- 1) 新增 sub_platform 列（与 ent schema field.String("sub_platform").Default("") 对齐）。
--    Postgres 11+ 下带常量默认值的加列为元数据操作，不会重写大表。
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS sub_platform VARCHAR(50) NOT NULL DEFAULT '';

-- 2) 将既有 Antigravity 账号迁移为 gemini + sub_platform='antigravity'。
--    覆盖软删除行（不加 deleted_at 过滤），保证口径一致；bump updated_at 以便
--    任何基于 updated_at 的缓存失效逻辑能感知变更。
UPDATE accounts
SET sub_platform = 'antigravity',
    platform = 'gemini',
    updated_at = NOW()
WHERE platform = 'antigravity';

-- 3) sub_platform 索引（与 ent schema index.Fields("sub_platform") 对齐），
--    服务于 Antigravity 专用路由/查询。
CREATE INDEX IF NOT EXISTS idx_accounts_sub_platform ON accounts (sub_platform);
