# SBP-DID Fabric Chaincode 设计说明书

## 一、项目简介
本项目基于原有EVM Solidity DID合约设计，重构为适用于Hyperledger Fabric的Golang Chaincode，实现DID、发证方、VC模板、VC存证等区块链身份与凭证管理功能。

---

## 二、模块结构

- **AccessControl 权限与配置管理**
- **DID 管理**
- **Issuer & VC模板管理**
- **VC存证管理**

所有模块集成在一个Chaincode主结构体中，模块间通过方法调用和状态存储实现数据与权限联动。

---

## 三、日志功能

### 日志作用
为了便于业务排查和问题定位，系统在关键业务位置添加了详细的日志记录。日志记录包括：
- 操作开始和结束状态
- 权限校验结果
- 参数校验结果
- 业务逻辑执行状态
- 数据存储操作结果
- 事件触发状态

### 日志记录内容
1. **AccessControl 模块日志**
   - 项目初始化配置日志
   - 权限校验成功/失败日志
   - 项目状态变更日志
   - 配置更新操作日志
   - 管理员权限转移日志
   - 批量权限操作日志

2. **DID 模块日志**
   - DID注册/更新操作日志
   - 权限校验结果日志
   - DID方法校验日志
   - 数据存储操作日志
   - 事件触发日志

3. **Issuer 模块日志**
   - 发证方注册/更新/状态变更日志
   - 审核状态检查日志
   - 权限校验结果日志
   - 数据唯一性校验日志
   - 业务操作结果日志

4. **VC 模块日志**
   - VC存证创建/查询/吊销日志
   - 发证方校验日志
   - 权限校验结果日志
   - 数据存储操作日志
   - 事件触发日志

### 日志使用方法
日志使用Go标准库的`log`包，通过`log.Printf`方法记录格式化的日志信息。日志会输出到链码的标准输出，可以通过Fabric的日志收集机制进行收集和分析。

---

## 四、项目结构

```shell
chaincode/
├── accesscontrol/
│   └── chaincode.go
├── did/
│   └── chaincode.go
├── issuer/
│   └── chaincode.go
├── vc/
│   └── chaincode.go
│
├── common/
│   └── utils.go         // 权限校验、事件封装等工具
├── main.go              // 初始化注册入口  
```

## 五、数据结构

### 1. AccessControl
```go
// 项目配置
 type ProjectConfig struct {
    IsPrivate                  bool   // 项目是否私有
    EnableVCTemplateVerification bool // 是否启用VC模板验证
    EnableIssuerVerification   bool   // 是否启用Issuer验证
    Method                     string // 项目method名称
    Paused                     bool   // 项目是否停用
}
// 账户权限
// map[账户地址]map[函数名]bool
```

### 2. DID管理
```go
type DidInfo struct {
    DidDocument string // DID文档
    Account     string // 注册账户
}
// map[DID]DidInfo
```

### 3. Issuer & VC模板管理
```go
type IssuerInfo struct {
    Name      string
    IsDisabled bool
    Account   string
}
type VcTemplateInfo struct {
    VcTemplateData string
    Account        string
    IsDisabled     bool
}
// map[issuerDid]IssuerInfo
// map[vcTemplateId]VcTemplateInfo
```

### 4. VC存证管理
```go
type VCInfo struct {
    VcId      string
    IssuerDid string
    VcHash    string
    IsRevoked bool
}
// map[vcId]VCInfo
```

---

## 六、主要方法与参数说明

### AccessControl
- InitProject(method, isPrivate, enableVC, enableIssuer)
- ChangePrivateStatus(isPrivate)
- ChangeMethod(method)
- ChangeEnableVCTemplateVerification(enable)
- ChangeEnableIssuerVerification(enable)
- BatchOperateSelectorPermissions([]SelectorPermission, isRevoke)
- HasSelectorPermission(account, selector) returns bool
- GetAllSelectorsForUser(account) returns []string
- Pause()/Unpause()
- IsProjectPrivate()/IsIssuerVerificationEnabled()/IsVCTemplateVerificationEnabled()/Paused() returns bool

### DID管理
- RegisterDid(did, didDocument)
- UpdateDidDocument(did, didDocument)
- GetDidInfo(did) returns didDocument
- CheckDid(did) returns bool

### Issuer & VC模板管理
- RegisterIssuer(issuerDid, name)
- UpdateIssuer(issuerDid, name)
- ChangeIssuerStatus(issuerDid, isDisabled)
- GetIssuerInfo(issuerDid) returns (name, isDisabled)
- CheckIssuer(issuerDid) returns bool
- RegisterVCTemplate(vcTemplateId, vcTemplateData)
- UpdateVCTemplate(vcTemplateId, vcTemplateData)
- ChangeVCTemplateStatus(vcTemplateId, isDisabled)
- GetVCTemplateInfo(vcTemplateId) returns (vcTemplateData, isDisabled)

### VC存证管理
- CreateVC(vcId, vcHash, issuerDid)
- GetVCHash(vcId) returns vcHash
- RevokeVC(vcId, isRevoked)
- GetVCRevokedStatus(vcId) returns isRevoked

---

## 七、调用流程与权限校验

- 管理员身份建议用Fabric的MSP（组织/用户证书）实现，或链上配置超级管理员账户。
- 其他权限用链上map存储，方法调用前先校验权限。
- 重要操作通过事件通知（stub.SetEvent）。
- 所有校验不通过时返回错误，终止交易。
- 所有关键操作都有详细的日志记录，便于问题排查。

---

## 八、日志排查指南

### 常见问题排查
1. **权限问题**：查看权限校验相关的日志，确认调用者身份和权限状态
2. **参数问题**：查看参数校验日志，确认输入参数的正确性
3. **业务逻辑问题**：查看业务执行流程日志，定位问题发生的具体步骤
4. **存储问题**：查看数据存储操作日志，确认数据写入状态
5. **事件问题**：查看事件触发日志，确认事件发送状态
6. Chaincode: Functions may only return a maximum of two values. GetIssuerInfo returns 3


### 日志分析技巧
- 通过调用者账户信息追踪特定用户的操作
- 通过操作类型和状态信息分析业务流程
- 通过错误信息快速定位问题原因
- 通过时间戳分析操作时序和性能

---

## 九、后续开发建议

- 按照本说明书实现Chaincode主结构体与各方法。
- 所有方法需完善注释，便于维护和二次开发。
- 可根据业务需要扩展更多DID、VC相关功能。
- 建议增加日志级别控制，支持生产环境的日志管理。
- 可以考虑增加结构化日志，便于日志分析和监控。

---

如需详细代码模板或具体实现，请联系开发负责人。 