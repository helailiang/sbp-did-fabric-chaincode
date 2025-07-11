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

## 项目结构

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

## 三、数据结构

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

## 四、主要方法与参数说明

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

## 五、调用流程与权限校验

- 管理员身份建议用Fabric的MSP（组织/用户证书）实现，或链上配置超级管理员账户。
- 其他权限用链上map存储，方法调用前先校验权限。
- 重要操作通过事件通知（stub.SetEvent）。
- 所有校验不通过时返回错误，终止交易。

---

## 六、后续开发建议

- 按照本说明书实现Chaincode主结构体与各方法。
- 所有方法需完善注释，便于维护和二次开发。
- 可根据业务需要扩展更多DID、VC相关功能。

---

如需详细代码模板或具体实现，请联系开发负责人。 