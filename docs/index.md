# 中医六经辨证开方文档

## 文档导航

### 核心文档

| 文档 | 说明 | 适用场景 |
|------|------|---------|
| [diagnosis_guide.md](diagnosis_guide.md) | 完整辨证指南（12步流程） | 系统学习、深入理解 |
| [quick_reference.md](quick_reference.md) | 速查表（表格化、一页式） | 临床快速查阅 |
| [checklist.md](checklist.md) | 辨证检查清单 | 临床逐项核对，避免漏项 |
| [flowchart.md](flowchart.md) | 辨证流程图 | 可视化理解流程和决策点 |
| [herbs/](herbs/index.md) | 中药材文档（三档分类体系） | 进药备货、存储管理、药证学习 |

### 方剂药证详解

#### 《伤寒论》方剂（112首 ✓ 基本完成）

| 目录 | 说明 | 完成数量 |
|------|------|----------|
| [太阳病方](formulas/shanghanlun/taiyang/index.md) | 解表类 | 22首 |
| [阳明病方](formulas/shanghanlun/yangming/index.md) | 清热攻下类 | 9首 |
| [少阳病方](formulas/shanghanlun/shaoyang/index.md) | 和解类 | 7首 |
| [太阴病方](formulas/shanghanlun/taiyin/index.md) | 温中类 | 7首 |
| [少阴病方](formulas/shanghanlun/shaoyin/index.md) | 回阳救逆类 | 20首 |
| [厥阴病方](formulas/shanghanlun/jueyin/index.md) | 寒热并用类 | 7首 |
| [其他方剂](formulas/shanghanlun/other/index.md) | 痞利水烦瘀等 | 40首 |
| [**总索引**](formulas/shanghanlun/index.md) | **《伤寒论》方剂总览** | **112首 ✓** |

#### 核心方剂速查

| 方剂 | 六经 | 证型 |
|------|------|------|
| [麻黄汤](formulas/shanghanlun/taiyang/mahuang_tang.md) | 太阳 | 表实证 |
| [桂枝汤](formulas/shanghanlun/taiyang/guizhi_tang.md) | 太阳 | 表虚证 |
| [白虎汤](formulas/shanghanlun/yangming/baihu_tang.md) | 阳明 | 经热 |
| [承气汤类](formulas/shanghanlun/yangming/chengqi_tang.md) | 阳明 | 腑实 |
| [小柴胡汤](formulas/shanghanlun/shaoyang/xiaochaihu_tang.md) | 少阳 | 枢机不利 |
| [理中汤](formulas/shanghanlun/taiyin/lizhong_tang.md) | 太阴 | 脾虚寒 |
| [四逆汤](formulas/shanghanlun/shaoyin/sini_tang.md) | 少阴 | 亡阳 |

### 案例库

| 案例 | 六经 | 方剂 | 说明 |
|------|------|------|------|
| [太阴脾虚寒案例](examples/taiyin_case.md) | 太阴 | 理中汤 | 完整12步演示 |
| [太阳表实证案例](examples/taiyang_case.md) | 待添加 | 麻黄汤 | — |
| [太阳表虚证案例](examples/taiyang_case.md) | 待添加 | 桂枝汤 | — |
| [阳明经热案例](examples/yangming_case.md) | 待添加 | 白虎汤 | — |
| [阳明腑实案例](examples/yangming_case.md) | 待添加 | 承气汤 | — |
| [少阳案例](examples/shaoyang_case.md) | 待添加 | 小柴胡汤 | — |
| [少阴案例](examples/shaoyin_case.md) | 待添加 | 四逆汤 | — |
| [厥阴案例](examples/jueyin_case.md) | 待添加 | 乌梅丸 | — |

---

## 快速入门

### 辨证流程概览

```
主诉病史 → 十问为纲 → 舌诊 → 脉诊
    ↓
定经（表里寒热虚实）→ 方证对勘 → 药证校验
    ↓
证据核查 → 反向验证 → 鉴别诊断 → 合病排查
    ↓
选方定药 → 剂量煎服法 → 医嘱
    ↓
复诊 → 评估 → 调整
```

### 核心原则

1. **有是证用是方** - 每个诊断都要有证据支撑
2. **双向验证** - 既找支持证据，也找矛盾点
3. **层层校验** - 问诊→舌脉→定经→方证→药证，每一步都是验证

---

## 学习建议

| 学习阶段 | 建议路径 |
|---------|---------|
| 初学者 | 先读 `diagnosis_guide.md` 理解流程 → 看 `flowchart.md` 可视化 → 再看案例库对照学习 |
| 有经验者 | 直接用 `quick_reference.md` 快速查阅 → 遇到疑难回看完整指南 |
| 临床实践 | 打印 `checklist.md` 随身携带 → 按12步流程记录病案 → 参考 `formulas/shanghanlun/` 深入理解方剂 |

---

## 文档结构

```
docs/
├── index.md                    ← 本文件（导航索引）
├── diagnosis_guide.md          ← 完整辨证指南（12步流程）
├── quick_reference.md          ← 速查表
├── checklist.md                ← 检查清单
├── flowchart.md                ← 流程图
├── formulas/
│   └── shanghanlun/            ← 《伤寒论》方剂
│       ├── index.md            ← 总索引
│       ├── taiyang/            ← 太阳病方
│       │   ├── index.md
│       │   ├── mahuang_tang.md
│       │   └── guizhi_tang.md
│       ├── yangming/           ← 阳明病方
│       │   ├── index.md
│       │   ├── baihu_tang.md
│       │   └── chengqi_tang.md
│       ├── shaoyang/           ← 少阳病方
│       │   ├── index.md
│       │   └── xiaochaihu_tang.md
│       ├── taiyin/             ← 太阴病方
│       │   ├── index.md
│       │   └── lizhong_tang.md
│       ├── shaoyin/            ← 少阴病方
│       │   ├── index.md
│       │   └── sini_tang.md
│       ├── jueyin/             ← 厥阴病方
│       │   └── index.md
│       └── other/              ← 其他方剂
│           └── index.md
└── examples/                   ← 案例库
    └── taiyin_case.md
```

---

## 版本信息

- 创建时间：2026-07-11
- 基础框架：六经辨证体系
- 方剂来源：《伤寒论》
- 适用范围：中医辨证开方入门与实践