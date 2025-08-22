package common

import (
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ProjectConfig 项目配置结构体定义
// 避免循环导入，直接定义结构体
type ProjectConfig struct {
	EnableVCTemplateVerification bool     `json:"enableVCTemplateVerification"` // 是否启用VC模板验证，启用后只有管理员可以管理模板
	EnableIssuerVerification     bool     `json:"enableIssuerVerification"`     // 是否启用Issuer验证，启用后只有管理员可以管理发证方
	EnableWritePermission        bool     `json:"enableWritePermission"`        // 是否开启合约写权限，控制普通用户是否可以进行写操作
	Method                       string   `json:"method"`                       // 项目method名称，用于DID标识符的method部分验证
	Paused                       bool     `json:"paused"`                       // 项目是否停用，停用后所有操作都会被拒绝
	IsProjectPrivate             bool     `json:"isProjectPrivate"`             // 项目是否私有，私有项目需要权限验证才能访问
	ServiceCode                  string   `json:"serviceCode"`                  // 服务编码，用于标识不同的服务实例
	ProjectCode                  string   `json:"projectCode"`                  // 项目编码，用于标识具体的项目
	Admins                       []string `json:"admins"`                       // 管理员账户地址SKI列表，具有最高权限
}

// PermissionChecker 权限检查接口
// 定义Permission模块需要实现的方法，供其他模块调用
type PermissionChecker interface {
	// 写权限检查
	CheckWriteFuncSelectorPermission(ctx contractapi.TransactionContextInterface, account, funcName string) (bool, error)

	// 查询权限检查
	CheckQueryFuncSelectorPermission(ctx contractapi.TransactionContextInterface, account, funcName string) (bool, error)

	// DID方法检查
	CheckMethod(ctx contractapi.TransactionContextInterface, did string) error

	// 检查账户是否可以进行颁发者（注册/更新等操作）
	CheckIssuerVerificationEnabled(ctx contractapi.TransactionContextInterface, account string) (bool, error)

	// VC模板验证检查
	CheckVCTemplateVerificationEnabled(ctx contractapi.TransactionContextInterface, account string) (bool, error)

	// 项目状态检查
	CheckNotPaused(ctx contractapi.TransactionContextInterface) error

	// 管理员角色检查
	CheckAdminRole(ctx contractapi.TransactionContextInterface, account string) error

	// 获取项目配置
	GetProjectConfig(ctx contractapi.TransactionContextInterface) (*ProjectConfig, error)
}

// GlobalPermissionChecker 全局权限检查器实例
// 在main.go中初始化，供所有模块使用
var GlobalPermissionChecker PermissionChecker

// SetGlobalPermissionChecker 设置全局权限检查器
func SetGlobalPermissionChecker(checker PermissionChecker) {
	GlobalPermissionChecker = checker
}

// GetGlobalPermissionChecker 获取全局权限检查器
func GetGlobalPermissionChecker() PermissionChecker {
	return GlobalPermissionChecker
}
