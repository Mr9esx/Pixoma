// Package apierr 定义 Pixoma 全部对外错误码。
//
// 错误码是 7 位整数：[HTTP:3][业务域:2][明细:2]。成功码固定为 2000000。
// 每个码对应唯一一条中文默认文案，前端按 code 查 apiError.<code> 覆盖语言。
// 本文件由 docs/api-error-messages.md 生成，不要手改。
package apierr

import (
	"sort"
	"strconv"
)

// SuccessCode 是所有成功响应的固定码。
const SuccessCode = 2000000

// Error 是一个对外错误。Code 是 7 位错误码，Message 是中文默认文案，
// I18nKey 是前端 i18n 词条键。
type Error struct {
	Code    int
	Message string
	I18nKey string
}

// Error 让 *Error 满足 error 接口，便于直接作为 error 返回。
func (e *Error) Error() string { return e.Message }

// I18nKeyFor 返回错误码对应的 i18n 词条键。
func I18nKeyFor(code int) string { return "apiError." + strconv.Itoa(code) }

var (
	// ErrSetupTestDatabaseDSNRequired — HTTP 400
	ErrSetupTestDatabaseDSNRequired = &Error{Code: 4000102, Message: "DSN 不能为空。填写数据库连接串后重试。", I18nKey: "apiError.4000102"}
	// ErrSetupFinalizeDefaultPasswordUnchanged — HTTP 400
	ErrSetupFinalizeDefaultPasswordUnchanged = &Error{Code: 4000110, Message: "仍在使用默认密码。先修改默认管理员密码。", I18nKey: "apiError.4000110"}
	// ErrSetupSaveDraftInvalid — HTTP 400
	ErrSetupSaveDraftInvalid = &Error{Code: 4000111, Message: "保存草稿失败。检查请求参数后重试。", I18nKey: "apiError.4000111"}
	// ErrSetupSaveSettingsInvalid — HTTP 400
	ErrSetupSaveSettingsInvalid = &Error{Code: 4000112, Message: "保存设置失败。检查请求参数后重试。", I18nKey: "apiError.4000112"}
	// ErrSetupSaveSettingsSetupNotFinalized — HTTP 400
	ErrSetupSaveSettingsSetupNotFinalized = &Error{Code: 4000113, Message: "初始化尚未结束。先运行完初始化向导。", I18nKey: "apiError.4000113"}
	// ErrSetupFinalizeInvalid — HTTP 400
	ErrSetupFinalizeInvalid = &Error{Code: 4000114, Message: "初始化收尾失败。检查请求参数后重试。", I18nKey: "apiError.4000114"}
	// ErrSetupChangePasswordPasswordAlreadySet — HTTP 400
	ErrSetupChangePasswordPasswordAlreadySet = &Error{Code: 4000115, Message: "密码已经设置。需要修改密码时使用「修改密码」。", I18nKey: "apiError.4000115"}
	// ErrSetupChangePasswordPasswordTooShort — HTTP 400
	ErrSetupChangePasswordPasswordTooShort = &Error{Code: 4000116, Message: "密码至少 8 位。增加长度后重试。", I18nKey: "apiError.4000116"}
	// ErrSetupTestBlobCheckFailed — HTTP 400
	ErrSetupTestBlobCheckFailed = &Error{Code: 4000117, Message: "对象存储连通失败。检查 Endpoint、Bucket 以及密钥后重新测试。", I18nKey: "apiError.4000117"}
	// ErrSetupTestDatabasePingFailed — HTTP 400
	ErrSetupTestDatabasePingFailed = &Error{Code: 4000118, Message: "数据库 ping 失败。检查 DSN 和网络后重新测试。", I18nKey: "apiError.4000118"}
	// ErrSetupSaveDraftDatabaseNotConfigured — HTTP 400
	ErrSetupSaveDraftDatabaseNotConfigured = &Error{Code: 4000119, Message: "数据库尚未配置。先在向导里填写 DSN。", I18nKey: "apiError.4000119"}
	// ErrSetupTestDatabaseUnreachable — HTTP 400
	ErrSetupTestDatabaseUnreachable = &Error{Code: 4000120, Message: "无法连接数据库。检查 DSN 和网络后重新测试。", I18nKey: "apiError.4000120"}
	// ErrSetupTestBlobConfigInvalid — HTTP 400
	ErrSetupTestBlobConfigInvalid = &Error{Code: 4000122, Message: "测试对象存储失败。检查请求参数后重试。", I18nKey: "apiError.4000122"}
	// ErrSetupTestDatabaseOpenFailed — HTTP 400
	ErrSetupTestDatabaseOpenFailed = &Error{Code: 4000123, Message: "测试数据库连接失败。检查请求参数后重试。", I18nKey: "apiError.4000123"}
	// ErrSetupSaveSettingsSettingsNotSaved — HTTP 400
	ErrSetupSaveSettingsSettingsNotSaved = &Error{Code: 4000125, Message: "设置尚未保存。先保存设置后继续。", I18nKey: "apiError.4000125"}
	// ErrSetupLoginInvalidJSON — HTTP 400
	ErrSetupLoginInvalidJSON = &Error{Code: 4000126, Message: "请求体格式不正确。检查字段格式后重试。", I18nKey: "apiError.4000126"}
	// ErrSetupLoadSettingsInvalid — HTTP 400
	ErrSetupLoadSettingsInvalid = &Error{Code: 4000127, Message: "读取设置失败。检查请求参数后重试。", I18nKey: "apiError.4000127"}
	// ErrSetupRegisterAccountNameRequired — HTTP 400
	ErrSetupRegisterAccountNameRequired = &Error{Code: 4000128, Message: "账号名不能为空。填写账号名后重试。", I18nKey: "apiError.4000128"}
	// ErrSetupSaveProfileInvalidEmail — HTTP 400
	ErrSetupSaveProfileInvalidEmail = &Error{Code: 4000129, Message: "邮箱格式不正确。检查后重试。", I18nKey: "apiError.4000129"}
	// ErrUserUpdateAccessInvalidAccess — HTTP 400
	ErrUserUpdateAccessInvalidAccess = &Error{Code: 4000202, Message: "访问权限不合法。检查后重试。", I18nKey: "apiError.4000202"}
	// ErrUserUpdateAccessInvalidJSON — HTTP 400
	ErrUserUpdateAccessInvalidJSON = &Error{Code: 4000211, Message: "请求体格式不正确。检查字段格式后重试。", I18nKey: "apiError.4000211"}
	// ErrUserListInvalidQuery — HTTP 400
	ErrUserListInvalidQuery = &Error{Code: 4000212, Message: "读取用户失败。检查请求参数后重试。", I18nKey: "apiError.4000212"}
	// ErrAdminUserListInvalidLimit — HTTP 400
	ErrAdminUserListInvalidLimit = &Error{Code: 4000302, Message: "`limit` 不合法。按照文档的取值范围填写。", I18nKey: "apiError.4000302"}
	// ErrAdminUserCreatePasswordTooShort — HTTP 400
	ErrAdminUserCreatePasswordTooShort = &Error{Code: 4000310, Message: "密码至少 8 位。增加长度后重试。", I18nKey: "apiError.4000310"}
	// ErrAdminUserUpdateInvalidRole — HTTP 400
	ErrAdminUserUpdateInvalidRole = &Error{Code: 4000311, Message: "角色不合法。只能填写 `admin` / `operator` / `viewer`。", I18nKey: "apiError.4000311"}
	// ErrAdminUserCreateInvalidJSON — HTTP 400
	ErrAdminUserCreateInvalidJSON = &Error{Code: 4000312, Message: "请求体格式不正确。检查字段格式后重试。", I18nKey: "apiError.4000312"}
	// ErrAdminUserCreateAccountNameRequired — HTTP 400
	ErrAdminUserCreateAccountNameRequired = &Error{Code: 4000313, Message: "账号名不能为空。填写账号名后重试。", I18nKey: "apiError.4000313"}
	// ErrAdminUserCreateInvalidEmail — HTTP 400
	ErrAdminUserCreateInvalidEmail = &Error{Code: 4000314, Message: "邮箱格式不正确。检查后重试。", I18nKey: "apiError.4000314"}
	// ErrSessionListInvalidQuery — HTTP 400
	ErrSessionListInvalidQuery = &Error{Code: 4000402, Message: "读取会话失败。检查请求参数后重试。", I18nKey: "apiError.4000402"}
	// ErrTaskListInvalidQuery — HTTP 400
	ErrTaskListInvalidQuery = &Error{Code: 4000502, Message: "读取任务失败。检查请求参数后重试。", I18nKey: "apiError.4000502"}
	// ErrCaseCreateInvalid — HTTP 400
	ErrCaseCreateInvalid = &Error{Code: 4000602, Message: "创建用例失败。检查请求参数后重试。", I18nKey: "apiError.4000602"}
	// ErrCaseUpdateInvalid — HTTP 400
	ErrCaseUpdateInvalid = &Error{Code: 4000610, Message: "更新用例失败。检查请求参数后重试。", I18nKey: "apiError.4000610"}
	// ErrCaseGetInvalidCaseID — HTTP 400
	ErrCaseGetInvalidCaseID = &Error{Code: 4000611, Message: "用例 ID 必须是数字。检查 URL 后重试。", I18nKey: "apiError.4000611"}
	// ErrCaseCreateInvalidJSON — HTTP 400
	ErrCaseCreateInvalidJSON = &Error{Code: 4000612, Message: "请求体格式不正确。检查字段格式后重试。", I18nKey: "apiError.4000612"}
	// ErrCaseListInvalidQuery — HTTP 400
	ErrCaseListInvalidQuery = &Error{Code: 4000613, Message: "读取用例失败。检查请求参数后重试。", I18nKey: "apiError.4000613"}
	// ErrEdgeListTasksInvalidLimit — HTTP 400
	ErrEdgeListTasksInvalidLimit = &Error{Code: 4000702, Message: "`limit` 不合法。按照文档的取值范围填写。", I18nKey: "apiError.4000702"}
	// ErrEdgeListTasksInvalidOffset — HTTP 400
	ErrEdgeListTasksInvalidOffset = &Error{Code: 4000710, Message: "`offset` 不合法。填写非负整数。", I18nKey: "apiError.4000710"}
	// ErrEdgeCreateNameRequired — HTTP 400
	ErrEdgeCreateNameRequired = &Error{Code: 4000711, Message: "名称不能为空。填写名称后重试。", I18nKey: "apiError.4000711"}
	// ErrEdgeUpdateTopicUnknown — HTTP 400
	ErrEdgeUpdateTopicUnknown = &Error{Code: 4000712, Message: "话题不存在或已停用。刷新话题列表后重试。", I18nKey: "apiError.4000712"}
	// ErrEdgeCreateInvalidJSON — HTTP 400
	ErrEdgeCreateInvalidJSON = &Error{Code: 4000713, Message: "请求体格式不正确。检查字段格式后重试。", I18nKey: "apiError.4000713"}
	// ErrEdgeMetricsInvalidQuery — HTTP 400
	ErrEdgeMetricsInvalidQuery = &Error{Code: 4000714, Message: "读取实例指标失败。检查请求参数后重试。", I18nKey: "apiError.4000714"}
	// ErrChannelCreateInvalid — HTTP 400
	ErrChannelCreateInvalid = &Error{Code: 4000802, Message: "创建渠道失败。检查请求参数后重试。", I18nKey: "apiError.4000802"}
	// ErrChannelCreateMCPUserNameRequired — HTTP 400
	ErrChannelCreateMCPUserNameRequired = &Error{Code: 4000811, Message: "名称不能为空。填写名称后重试。", I18nKey: "apiError.4000811"}
	// ErrChannelPutMenuInvalid — HTTP 400
	ErrChannelPutMenuInvalid = &Error{Code: 4000812, Message: "处理渠道失败。检查请求参数后重试。", I18nKey: "apiError.4000812"}
	// ErrChannelCreateMCPUserNotMCP — HTTP 400
	ErrChannelCreateMCPUserNotMCP = &Error{Code: 4000813, Message: "渠道类型不支持。更换为 MCP 类型的渠道后重试。", I18nKey: "apiError.4000813"}
	// ErrChannelCreateInvalidJSON — HTTP 400
	ErrChannelCreateInvalidJSON = &Error{Code: 4000814, Message: "请求体格式不正确。检查字段格式后重试。", I18nKey: "apiError.4000814"}
	// ErrTopicCreateNameRequired — HTTP 400
	ErrTopicCreateNameRequired = &Error{Code: 4000902, Message: "名称不能为空。填写名称后重试。", I18nKey: "apiError.4000902"}
	// ErrTopicCreateInvalid — HTTP 400
	ErrTopicCreateInvalid = &Error{Code: 4000910, Message: "话题 key 不合法。仅支持小写字母、数字和连字符。", I18nKey: "apiError.4000910"}
	// ErrTopicCreateInvalidJSON — HTTP 400
	ErrTopicCreateInvalidJSON = &Error{Code: 4000911, Message: "请求体格式不正确。检查字段格式后重试。", I18nKey: "apiError.4000911"}
	// ErrStatsErrorsInvalidLimit — HTTP 400
	ErrStatsErrorsInvalidLimit = &Error{Code: 4001102, Message: "`limit` 超出范围。填写 1–100。", I18nKey: "apiError.4001102"}
	// ErrStatsCasesTopInvalidLimit — HTTP 400
	ErrStatsCasesTopInvalidLimit = &Error{Code: 4001110, Message: "`limit` 超出范围。填写 1–20。", I18nKey: "apiError.4001110"}
	// ErrStatsRangeReversed — HTTP 400
	ErrStatsRangeReversed = &Error{Code: 4001111, Message: "开始日期晚于结束日期。调整后重试。", I18nKey: "apiError.4001111"}
	// ErrStatsInvalidRange — HTTP 400
	ErrStatsInvalidRange = &Error{Code: 4001112, Message: "日期格式不正确。使用 YYYY-MM-DD 后重试。", I18nKey: "apiError.4001112"}
	// ErrStatsRangeTooLong — HTTP 400
	ErrStatsRangeTooLong = &Error{Code: 4001113, Message: "查询范围超过 365 天。缩小范围后重试。", I18nKey: "apiError.4001113"}
	// ErrStudioStreamAGUIMissingFields — HTTP 400
	ErrStudioStreamAGUIMissingFields = &Error{Code: 4001202, Message: "会话、运行、消息都不能为空。补充完整后重试。", I18nKey: "apiError.4001202"}
	// ErrStudioUploadAssetFileMissing — HTTP 400
	ErrStudioUploadAssetFileMissing = &Error{Code: 4001212, Message: "尚未选择文件。选择文件后重新上传。", I18nKey: "apiError.4001212"}
	// ErrStudioStreamAGUIInvalidJSON — HTTP 400
	ErrStudioStreamAGUIInvalidJSON = &Error{Code: 4001213, Message: "请求体格式不正确。检查字段格式后重试。", I18nKey: "apiError.4001213"}
	// ErrStudioInvalidBody — HTTP 400
	ErrStudioInvalidBody = &Error{Code: 4001214, Message: "请求内容不合法。检查后重新提交。", I18nKey: "apiError.4001214"}
	// ErrStudioUploadAssetFileReadFailed — HTTP 400
	ErrStudioUploadAssetFileReadFailed = &Error{Code: 4001215, Message: "读取上传文件失败。重新选择文件后上传。", I18nKey: "apiError.4001215"}
	// ErrMediaUploadFileMissing — HTTP 400
	ErrMediaUploadFileMissing = &Error{Code: 4001302, Message: "尚未选择文件。选择文件后重新上传。", I18nKey: "apiError.4001302"}
	// ErrMediaUploadFileTypeUnsupported — HTTP 400
	ErrMediaUploadFileTypeUnsupported = &Error{Code: 4001310, Message: "文件类型不支持。仅支持 png / jpeg / webp / gif 图片以及 mp4 / webm 视频。", I18nKey: "apiError.4001310"}
	// ErrMediaUploadInvalidContentType — HTTP 400
	ErrMediaUploadInvalidContentType = &Error{Code: 4001312, Message: "请求格式不支持。使用 multipart/form-data 重新上传。", I18nKey: "apiError.4001312"}
	// ErrMediaUploadFileReadFailed — HTTP 400
	ErrMediaUploadFileReadFailed = &Error{Code: 4001313, Message: "读取上传文件失败。重新选择文件后上传。", I18nKey: "apiError.4001313"}
	// ErrAgentClaimEdgeIDRequired — HTTP 400
	ErrAgentClaimEdgeIDRequired = &Error{Code: 4001502, Message: "`edge_id` 不能为空。携带实例 ID 后重试。", I18nKey: "apiError.4001502"}
	// ErrAgentPresenceInvalidJSON — HTTP 400
	ErrAgentPresenceInvalidJSON = &Error{Code: 4001510, Message: "请求体格式不正确。检查上报字段的格式。", I18nKey: "apiError.4001510"}
	// ErrAgentHeartbeatInvalidJSON — HTTP 400
	ErrAgentHeartbeatInvalidJSON = &Error{Code: 4001511, Message: "请求体格式不正确。检查字段格式后重试。", I18nKey: "apiError.4001511"}

	// ErrCommonPermissionUnauthorized — HTTP 401
	ErrCommonPermissionUnauthorized = &Error{Code: 4010007, Message: "登录已失效。重新登录。", I18nKey: "apiError.4010007"}
	// ErrSetupSessionUnauthorized — HTTP 401
	ErrSetupSessionUnauthorized = &Error{Code: 4010107, Message: "登录已失效。重新登录。", I18nKey: "apiError.4010107"}
	// ErrSetupLoginUnauthorized — HTTP 401
	ErrSetupLoginUnauthorized = &Error{Code: 4010140, Message: "账号或密码不正确。检查后重试。", I18nKey: "apiError.4010140"}
	// ErrStudioAccountUnauthorized — HTTP 401
	ErrStudioAccountUnauthorized = &Error{Code: 4011207, Message: "登录已失效。重新登录。", I18nKey: "apiError.4011207"}
	// ErrAgentClaimUnauthorized — HTTP 401
	ErrAgentClaimUnauthorized = &Error{Code: 4011507, Message: "凭证无效。检查实例的 `agent token`。", I18nKey: "apiError.4011507"}

	// ErrCommonPermissionForbidden — HTTP 403
	ErrCommonPermissionForbidden = &Error{Code: 4030008, Message: "没有权限。这个操作需要管理员。", I18nKey: "apiError.4030008"}
	// ErrSetupNotInitialized — HTTP 403
	ErrSetupNotInitialized = &Error{Code: 4030108, Message: "平台尚未初始化。先完成初始化设置。", I18nKey: "apiError.4030108"}
	// ErrSetupSessionForbidden — HTTP 403
	ErrSetupSessionForbidden = &Error{Code: 4030141, Message: "没有权限。这个操作需要管理员。", I18nKey: "apiError.4030141"}
	// ErrSetupRestartRequired — HTTP 403
	ErrSetupRestartRequired = &Error{Code: 4030142, Message: "设置已保存。重启 `pixoma` 后生效。", I18nKey: "apiError.4030142"}
	// ErrSetupAccountDisabled — HTTP 403
	ErrSetupAccountDisabled = &Error{Code: 4030143, Message: "这个账号已停用。联系管理员启用。", I18nKey: "apiError.4030143"}
	// ErrUserRotateMCPTokenForbidden — HTTP 403
	ErrUserRotateMCPTokenForbidden = &Error{Code: 4030208, Message: "没有权限。这个操作需要管理员。", I18nKey: "apiError.4030208"}

	// ErrSetupBlobBucketMissing — HTTP 404
	ErrSetupBlobBucketMissing = &Error{Code: 4040101, Message: "bucket 不存在。勾选「自动创建」后重新测试。", I18nKey: "apiError.4040101"}
	// ErrUserLoadMCPTokenNotFound — HTTP 404
	ErrUserLoadMCPTokenNotFound = &Error{Code: 4040201, Message: "MCP token 不存在。先在用户详情里生成一个。", I18nKey: "apiError.4040201"}
	// ErrUserGetNotFound — HTTP 404
	ErrUserGetNotFound = &Error{Code: 4040210, Message: "用户不存在。可能已被删除，刷新列表后重试。", I18nKey: "apiError.4040210"}
	// ErrAdminUserNotFound — HTTP 404
	ErrAdminUserNotFound = &Error{Code: 4040301, Message: "管理员不存在。可能已被删除，刷新列表后重试。", I18nKey: "apiError.4040301"}
	// ErrSessionGetNotFound — HTTP 404
	ErrSessionGetNotFound = &Error{Code: 4040401, Message: "会话不存在。可能已被删除，刷新列表后重试。", I18nKey: "apiError.4040401"}
	// ErrTaskGetNotFound — HTTP 404
	ErrTaskGetNotFound = &Error{Code: 4040501, Message: "任务不存在。可能已被删除，刷新列表后重试。", I18nKey: "apiError.4040501"}
	// ErrCaseGetNotFound — HTTP 404
	ErrCaseGetNotFound = &Error{Code: 4040601, Message: "用例不存在。可能已被删除，刷新列表后重试。", I18nKey: "apiError.4040601"}
	// ErrEdgeGetNotFound — HTTP 404
	ErrEdgeGetNotFound = &Error{Code: 4040701, Message: "实例不存在。可能已被删除，刷新列表后重试。", I18nKey: "apiError.4040701"}
	// ErrChannelGetNotFound — HTTP 404
	ErrChannelGetNotFound = &Error{Code: 4040801, Message: "渠道不存在。可能已被删除，刷新列表后重试。", I18nKey: "apiError.4040801"}
	// ErrTopicGetNotFound — HTTP 404
	ErrTopicGetNotFound = &Error{Code: 4040901, Message: "话题不存在。可能已被删除，刷新列表后重试。", I18nKey: "apiError.4040901"}
	// ErrStudioNotFound — HTTP 404
	ErrStudioNotFound = &Error{Code: 4041201, Message: "内容不存在或无权访问。刷新列表后重试。", I18nKey: "apiError.4041201"}
	// ErrStudioAssetNotFound — HTTP 404
	ErrStudioAssetNotFound = &Error{Code: 4041210, Message: "资产内容不存在。刷新素材库后重试。", I18nKey: "apiError.4041210"}
	// ErrStudioAssetVersionNotFound — HTTP 404
	ErrStudioAssetVersionNotFound = &Error{Code: 4041211, Message: "这个资产版本不存在。刷新后选择其他版本。", I18nKey: "apiError.4041211"}
	// ErrMediaServePreviewNotFound — HTTP 404
	ErrMediaServePreviewNotFound = &Error{Code: 4041301, Message: "媒体不存在。可能已被删除，刷新后重试。", I18nKey: "apiError.4041301"}

	// ErrAgentClaimCanceled — HTTP 408
	ErrAgentClaimCanceled = &Error{Code: 4081512, Message: "请求被取消。重新领取任务。", I18nKey: "apiError.4081512"}

	// ErrSetupRegisterRegistrationDisabled — HTTP 409
	ErrSetupRegisterRegistrationDisabled = &Error{Code: 4090103, Message: "自助注册已关闭。需要管理员在后台开启。", I18nKey: "apiError.4090103"}
	// ErrSetupRegisterAlreadyTaken — HTTP 409
	ErrSetupRegisterAlreadyTaken = &Error{Code: 4090130, Message: "账号名或邮箱已被占用。更换后重试。", I18nKey: "apiError.4090130"}
	// ErrAdminUserUpdateLastAdmin — HTTP 409
	ErrAdminUserUpdateLastAdmin = &Error{Code: 4090303, Message: "只剩这一个管理员。先添加管理员后再修改。", I18nKey: "apiError.4090303"}
	// ErrAdminUserDeleteLastAdmin — HTTP 409
	ErrAdminUserDeleteLastAdmin = &Error{Code: 4090315, Message: "只剩这一个管理员。先添加管理员后再删除。", I18nKey: "apiError.4090315"}
	// ErrAdminUserAlreadyTaken — HTTP 409
	ErrAdminUserAlreadyTaken = &Error{Code: 4090316, Message: "账号名或邮箱已被占用。更换后重试。", I18nKey: "apiError.4090316"}
	// ErrTaskCancelCancelNotAllowed — HTTP 409
	ErrTaskCancelCancelNotAllowed = &Error{Code: 4090503, Message: "当前状态不能取消。等待任务运行结束后重试。", I18nKey: "apiError.4090503"}
	// ErrCaseCreateAlreadyExists — HTTP 409
	ErrCaseCreateAlreadyExists = &Error{Code: 4090603, Message: "用例已存在。修改名称，或者先删除原有的。", I18nKey: "apiError.4090603"}
	// ErrCaseDeleteConflict — HTTP 409
	ErrCaseDeleteConflict = &Error{Code: 4090604, Message: "用例被菜单或卡片引用。勾选「同时清理引用」后删除。", I18nKey: "apiError.4090604"}
	// ErrEdgeCreateAlreadyExists — HTTP 409
	ErrEdgeCreateAlreadyExists = &Error{Code: 4090703, Message: "实例已存在。修改名称，或者先删除原有的。", I18nKey: "apiError.4090703"}
	// ErrEdgeDeleteConflict — HTTP 409
	ErrEdgeDeleteConflict = &Error{Code: 4090704, Message: "实例还有运行中的任务。勾选「同时清理引用」后删除，任务会被标记为失败。", I18nKey: "apiError.4090704"}
	// ErrTopicCreateAlreadyExists — HTTP 409
	ErrTopicCreateAlreadyExists = &Error{Code: 4090903, Message: "话题 key 已存在。更换一个 key。", I18nKey: "apiError.4090903"}
	// ErrTopicDeleteConflict — HTTP 409
	ErrTopicDeleteConflict = &Error{Code: 4090904, Message: "话题被用例或实例引用。勾选「同时清理引用」后删除。", I18nKey: "apiError.4090904"}
	// ErrTopicUpdateDefaultProtected — HTTP 409
	ErrTopicUpdateDefaultProtected = &Error{Code: 4090912, Message: "默认话题不能停用。更换其他话题后重试。", I18nKey: "apiError.4090912"}
	// ErrTopicDeleteDefaultProtected — HTTP 409
	ErrTopicDeleteDefaultProtected = &Error{Code: 4090913, Message: "默认话题不能删除。只能删除自建话题。", I18nKey: "apiError.4090913"}
	// ErrStudioAlreadyExists — HTTP 409
	ErrStudioAlreadyExists = &Error{Code: 4091203, Message: "内容已存在。修改名称，或者先删除原有的。", I18nKey: "apiError.4091203"}
	// ErrAgentStatusConflict — HTTP 409
	ErrAgentStatusConflict = &Error{Code: 4091503, Message: "上报任务状态失败。刷新后重试。", I18nKey: "apiError.4091503"}
	// ErrAgentStatusTaskNotFound — HTTP 409
	ErrAgentStatusTaskNotFound = &Error{Code: 4091513, Message: "任务不存在。可能已被回收，重新领取任务。", I18nKey: "apiError.4091513"}
	// ErrAgentStatusStaleHolder — HTTP 409
	ErrAgentStatusStaleHolder = &Error{Code: 4091514, Message: "任务持有者已过期。重新领取任务后上报。", I18nKey: "apiError.4091514"}
	// ErrAgentHeartbeatHeartbeatRejected — HTTP 409
	ErrAgentHeartbeatHeartbeatRejected = &Error{Code: 4091515, Message: "心跳被拒绝。任务可能已被回收，重新领取任务。", I18nKey: "apiError.4091515"}

	// ErrChannelGoneCardsUnknown — HTTP 410
	ErrChannelGoneCardsUnknown = &Error{Code: 4100810, Message: "渠道不存在。可能已被删除，刷新列表后重试。", I18nKey: "apiError.4100810"}

	// ErrCommonRequestBodyTooLarge — HTTP 413
	ErrCommonRequestBodyTooLarge = &Error{Code: 4130002, Message: "请求内容超过大小上限。缩小后重新提交。", I18nKey: "apiError.4130002"}
	// ErrMediaUploadTooLarge — HTTP 413
	ErrMediaUploadTooLarge = &Error{Code: 4131311, Message: "文件超过大小上限。压缩后重新上传。", I18nKey: "apiError.4131311"}

	// ErrSetupRegisterTooManyRequests — HTTP 429
	ErrSetupRegisterTooManyRequests = &Error{Code: 4290121, Message: "注册请求过多。等待 1 分钟后重试。", I18nKey: "apiError.4290121"}
	// ErrSetupLoginTooManyRequests — HTTP 429
	ErrSetupLoginTooManyRequests = &Error{Code: 4290124, Message: "登录尝试过多。等待 1 分钟后重试。", I18nKey: "apiError.4290124"}

	// ErrCommonInternal — HTTP 500
	ErrCommonInternal = &Error{Code: 5000005, Message: "服务暂时不可用。重启 `pixoma` 后重试。", I18nKey: "apiError.5000005"}
	// ErrSetupRegisterServiceUnavailable — HTTP 500
	ErrSetupRegisterServiceUnavailable = &Error{Code: 5000105, Message: "账号服务不可用。重启 `pixoma` 后重试。", I18nKey: "apiError.5000105"}
	// ErrSetupSaveDraftFailed — HTTP 500
	ErrSetupSaveDraftFailed = &Error{Code: 5000106, Message: "保存草稿失败。重新保存一次。", I18nKey: "apiError.5000106"}
	// ErrSetupSaveSettingsFailed — HTTP 500
	ErrSetupSaveSettingsFailed = &Error{Code: 5000131, Message: "保存设置失败。重新保存一次。", I18nKey: "apiError.5000131"}
	// ErrSetupSaveProfileFailed — HTTP 500
	ErrSetupSaveProfileFailed = &Error{Code: 5000132, Message: "保存资料失败。重新保存一次。", I18nKey: "apiError.5000132"}
	// ErrSetupChangePasswordFailed — HTTP 500
	ErrSetupChangePasswordFailed = &Error{Code: 5000133, Message: "修改密码失败。重新提交一次。", I18nKey: "apiError.5000133"}
	// ErrSetupRegisterCreateFailed — HTTP 500
	ErrSetupRegisterCreateFailed = &Error{Code: 5000134, Message: "创建账号失败。重新提交一次。", I18nKey: "apiError.5000134"}
	// ErrSetupFinalizeFailed — HTTP 500
	ErrSetupFinalizeFailed = &Error{Code: 5000135, Message: "初始化收尾失败。重新提交一次。", I18nKey: "apiError.5000135"}
	// ErrSetupRegisterHashPasswordFailed — HTTP 500
	ErrSetupRegisterHashPasswordFailed = &Error{Code: 5000136, Message: "密码加密失败。重新提交一次。", I18nKey: "apiError.5000136"}
	// ErrSetupRegisterFailed — HTTP 500
	ErrSetupRegisterFailed = &Error{Code: 5000137, Message: "注册失败。重新提交一次。", I18nKey: "apiError.5000137"}
	// ErrSetupTestDatabaseFailed — HTTP 500
	ErrSetupTestDatabaseFailed = &Error{Code: 5000138, Message: "测试数据库连接失败。检查 DSN 和网络后重新测试。", I18nKey: "apiError.5000138"}
	// ErrSetupLoginFailed — HTTP 500
	ErrSetupLoginFailed = &Error{Code: 5000139, Message: "登录失败。等待 1 分钟后重试。", I18nKey: "apiError.5000139"}
	// ErrUserLoadMCPTokenNotConfigured — HTTP 500
	ErrUserLoadMCPTokenNotConfigured = &Error{Code: 5000206, Message: "MCP token 还没生成。先在用户详情里生成。", I18nKey: "apiError.5000206"}
	// ErrUserUpdateAccessFailed — HTTP 500
	ErrUserUpdateAccessFailed = &Error{Code: 5000213, Message: "更新访问权限失败。重新保存一次。", I18nKey: "apiError.5000213"}
	// ErrUserGetMCPTokenFailed — HTTP 500
	ErrUserGetMCPTokenFailed = &Error{Code: 5000214, Message: "读取 MCP token 失败。刷新页面后重试。", I18nKey: "apiError.5000214"}
	// ErrUserListFailed — HTTP 500
	ErrUserListFailed = &Error{Code: 5000215, Message: "读取用户失败。刷新页面后重试。", I18nKey: "apiError.5000215"}
	// ErrUserRotateMCPTokenFailed — HTTP 500
	ErrUserRotateMCPTokenFailed = &Error{Code: 5000216, Message: "轮换 MCP token 失败。重新生成一次。", I18nKey: "apiError.5000216"}
	// ErrAdminUserCreateFailed — HTTP 500
	ErrAdminUserCreateFailed = &Error{Code: 5000306, Message: "创建管理员失败。重新提交一次。", I18nKey: "apiError.5000306"}
	// ErrAdminUserDeleteDeleteFailed — HTTP 500
	ErrAdminUserDeleteDeleteFailed = &Error{Code: 5000317, Message: "删除管理员失败。重试一次。", I18nKey: "apiError.5000317"}
	// ErrAdminUserCreateHashPasswordFailed — HTTP 500
	ErrAdminUserCreateHashPasswordFailed = &Error{Code: 5000318, Message: "密码加密失败。重新提交一次。", I18nKey: "apiError.5000318"}
	// ErrAdminUserUpdateFailed — HTTP 500
	ErrAdminUserUpdateFailed = &Error{Code: 5000319, Message: "更新管理员失败。重新保存一次。", I18nKey: "apiError.5000319"}
	// ErrAdminUserUpdateCountAdminsFailed — HTTP 500
	ErrAdminUserUpdateCountAdminsFailed = &Error{Code: 5000320, Message: "统计管理员数量失败。刷新页面后重试。", I18nKey: "apiError.5000320"}
	// ErrAdminUserListListFailed — HTTP 500
	ErrAdminUserListListFailed = &Error{Code: 5000321, Message: "读取管理员列表失败。刷新页面后重试。", I18nKey: "apiError.5000321"}
	// ErrAdminUserLoadFailed — HTTP 500
	ErrAdminUserLoadFailed = &Error{Code: 5000322, Message: "读取管理员失败。刷新页面后重试。", I18nKey: "apiError.5000322"}
	// ErrSessionListFailed — HTTP 500
	ErrSessionListFailed = &Error{Code: 5000406, Message: "读取会话失败。刷新页面后重试。", I18nKey: "apiError.5000406"}
	// ErrTaskCancelNotConfigured — HTTP 500
	ErrTaskCancelNotConfigured = &Error{Code: 5000505, Message: "任务取消服务未配置。检查配置后重启 `pixoma`。", I18nKey: "apiError.5000505"}
	// ErrTaskCancelFailed — HTTP 500
	ErrTaskCancelFailed = &Error{Code: 5000506, Message: "取消任务失败。重新提交一次。", I18nKey: "apiError.5000506"}
	// ErrTaskListFailed — HTTP 500
	ErrTaskListFailed = &Error{Code: 5000510, Message: "读取任务失败。刷新页面后重试。", I18nKey: "apiError.5000510"}
	// ErrCaseDeleteNotConfigured — HTTP 500
	ErrCaseDeleteNotConfigured = &Error{Code: 5000605, Message: "删除清理服务未配置。检查配置后重启 `pixoma`。", I18nKey: "apiError.5000605"}
	// ErrCaseDisableFailed — HTTP 500
	ErrCaseDisableFailed = &Error{Code: 5000606, Message: "停用用例失败。重试一次。", I18nKey: "apiError.5000606"}
	// ErrCaseCreateFailed — HTTP 500
	ErrCaseCreateFailed = &Error{Code: 5000614, Message: "创建用例失败。重新提交一次。", I18nKey: "apiError.5000614"}
	// ErrCaseDeleteFailed — HTTP 500
	ErrCaseDeleteFailed = &Error{Code: 5000615, Message: "删除用例失败。重试一次。", I18nKey: "apiError.5000615"}
	// ErrCaseEnableFailed — HTTP 500
	ErrCaseEnableFailed = &Error{Code: 5000616, Message: "启用用例失败。重试一次。", I18nKey: "apiError.5000616"}
	// ErrCaseUpdateFailed — HTTP 500
	ErrCaseUpdateFailed = &Error{Code: 5000617, Message: "更新用例失败。重新保存一次。", I18nKey: "apiError.5000617"}
	// ErrCaseListFailed — HTTP 500
	ErrCaseListFailed = &Error{Code: 5000618, Message: "读取用例失败。刷新页面后重试。", I18nKey: "apiError.5000618"}
	// ErrEdgeDeleteNotConfigured — HTTP 500
	ErrEdgeDeleteNotConfigured = &Error{Code: 5000705, Message: "删除清理服务未配置。检查配置后重启 `pixoma`。", I18nKey: "apiError.5000705"}
	// ErrEdgeWriteFailed — HTTP 500
	ErrEdgeWriteFailed = &Error{Code: 5000706, Message: "写入实例失败。重试一次。", I18nKey: "apiError.5000706"}
	// ErrEdgeMetricsNotConfigured — HTTP 500
	ErrEdgeMetricsNotConfigured = &Error{Code: 5000715, Message: "指标服务未配置。检查 `metrics` 配置后重启 `pixoma`。", I18nKey: "apiError.5000715"}
	// ErrEdgeUpdateNotConfigured — HTTP 500
	ErrEdgeUpdateNotConfigured = &Error{Code: 5000716, Message: "话题存储未配置。检查配置后重启 `pixoma`。", I18nKey: "apiError.5000716"}
	// ErrEdgeCreateFailed — HTTP 500
	ErrEdgeCreateFailed = &Error{Code: 5000717, Message: "创建实例失败。重新提交一次。", I18nKey: "apiError.5000717"}
	// ErrEdgeDeleteFailed — HTTP 500
	ErrEdgeDeleteFailed = &Error{Code: 5000718, Message: "删除实例失败。重试一次。", I18nKey: "apiError.5000718"}
	// ErrEdgeUpdateFailed — HTTP 500
	ErrEdgeUpdateFailed = &Error{Code: 5000719, Message: "更新实例失败。重新保存一次。", I18nKey: "apiError.5000719"}
	// ErrEdgeListTasksFailed — HTTP 500
	ErrEdgeListTasksFailed = &Error{Code: 5000720, Message: "读取任务列表失败。刷新页面后重试。", I18nKey: "apiError.5000720"}
	// ErrEdgeListPresenceFailed — HTTP 500
	ErrEdgeListPresenceFailed = &Error{Code: 5000721, Message: "读取在线实例失败。刷新页面后重试。", I18nKey: "apiError.5000721"}
	// ErrEdgeListFailed — HTTP 500
	ErrEdgeListFailed = &Error{Code: 5000722, Message: "读取实例失败。刷新页面后重试。", I18nKey: "apiError.5000722"}
	// ErrEdgeMetricsFailed — HTTP 500
	ErrEdgeMetricsFailed = &Error{Code: 5000723, Message: "读取实例指标失败。刷新页面后重试。", I18nKey: "apiError.5000723"}
	// ErrEdgeStatsFailed — HTTP 500
	ErrEdgeStatsFailed = &Error{Code: 5000724, Message: "读取统计失败。刷新页面后重试。", I18nKey: "apiError.5000724"}
	// ErrEdgeRotateTokenFailed — HTTP 500
	ErrEdgeRotateTokenFailed = &Error{Code: 5000725, Message: "轮换实例 token 失败。重新生成一次。", I18nKey: "apiError.5000725"}
	// ErrChannelCreateMCPUserNotConfigured — HTTP 500
	ErrChannelCreateMCPUserNotConfigured = &Error{Code: 5000805, Message: "MCP 用户服务未配置。检查 `channels` 配置后重启 `pixoma`。", I18nKey: "apiError.5000805"}
	// ErrChannelDisableFailed — HTTP 500
	ErrChannelDisableFailed = &Error{Code: 5000806, Message: "停用渠道失败。重试一次。", I18nKey: "apiError.5000806"}
	// ErrChannelCreateNotConfigured — HTTP 500
	ErrChannelCreateNotConfigured = &Error{Code: 5000816, Message: "渠道服务未配置。检查 `channels` 配置后重启 `pixoma`。", I18nKey: "apiError.5000816"}
	// ErrChannelCreateFailed — HTTP 500
	ErrChannelCreateFailed = &Error{Code: 5000817, Message: "创建渠道失败。重新提交一次。", I18nKey: "apiError.5000817"}
	// ErrChannelDeleteMCPUserFailed — HTTP 500
	ErrChannelDeleteMCPUserFailed = &Error{Code: 5000818, Message: "删除 MCP 用户失败。重试一次。", I18nKey: "apiError.5000818"}
	// ErrChannelDeleteFailed — HTTP 500
	ErrChannelDeleteFailed = &Error{Code: 5000819, Message: "删除渠道失败。重试一次。", I18nKey: "apiError.5000819"}
	// ErrChannelEnableFailed — HTTP 500
	ErrChannelEnableFailed = &Error{Code: 5000820, Message: "启用渠道失败。重试一次。", I18nKey: "apiError.5000820"}
	// ErrChannelGetMenuFailed — HTTP 500
	ErrChannelGetMenuFailed = &Error{Code: 5000821, Message: "处理渠道失败。重试一次。", I18nKey: "apiError.5000821"}
	// ErrChannelUpdateFailed — HTTP 500
	ErrChannelUpdateFailed = &Error{Code: 5000822, Message: "更新渠道失败。重新保存一次。", I18nKey: "apiError.5000822"}
	// ErrChannelCreateMCPUserFailed — HTTP 500
	ErrChannelCreateMCPUserFailed = &Error{Code: 5000823, Message: "添加 MCP 用户失败。重新提交一次。", I18nKey: "apiError.5000823"}
	// ErrChannelListMCPUsersFailed — HTTP 500
	ErrChannelListMCPUsersFailed = &Error{Code: 5000824, Message: "读取 MCP 用户失败。刷新页面后重试。", I18nKey: "apiError.5000824"}
	// ErrChannelListFailed — HTTP 500
	ErrChannelListFailed = &Error{Code: 5000825, Message: "读取渠道失败。刷新页面后重试。", I18nKey: "apiError.5000825"}
	// ErrTopicDeleteNotConfigured — HTTP 500
	ErrTopicDeleteNotConfigured = &Error{Code: 5000905, Message: "删除清理服务未配置。检查配置后重启 `pixoma`。", I18nKey: "apiError.5000905"}
	// ErrTopicCreateFailed — HTTP 500
	ErrTopicCreateFailed = &Error{Code: 5000906, Message: "创建话题失败。重新提交一次。", I18nKey: "apiError.5000906"}
	// ErrTopicDeleteFailed — HTTP 500
	ErrTopicDeleteFailed = &Error{Code: 5000914, Message: "删除话题失败。重试一次。", I18nKey: "apiError.5000914"}
	// ErrTopicUpdateFailed — HTTP 500
	ErrTopicUpdateFailed = &Error{Code: 5000915, Message: "更新话题失败。重新保存一次。", I18nKey: "apiError.5000915"}
	// ErrTopicStatsFailed — HTTP 500
	ErrTopicStatsFailed = &Error{Code: 5000916, Message: "读取统计失败。刷新页面后重试。", I18nKey: "apiError.5000916"}
	// ErrTopicListFailed — HTTP 500
	ErrTopicListFailed = &Error{Code: 5000917, Message: "读取话题失败。刷新页面后重试。", I18nKey: "apiError.5000917"}
	// ErrRoutingAttributesNotConfigured — HTTP 500
	ErrRoutingAttributesNotConfigured = &Error{Code: 5001005, Message: "路由注册表未配置。检查配置后重启 `pixoma`。", I18nKey: "apiError.5001005"}
	// ErrStatsFleetNotConfigured — HTTP 500
	ErrStatsFleetNotConfigured = &Error{Code: 5001105, Message: "指标服务未配置。检查 `metrics` 配置后重启 `pixoma`。", I18nKey: "apiError.5001105"}
	// ErrStatsEdgesFailed — HTTP 500
	ErrStatsEdgesFailed = &Error{Code: 5001106, Message: "读取实例统计失败。刷新页面后重试。", I18nKey: "apiError.5001106"}
	// ErrStatsDailyFailed — HTTP 500
	ErrStatsDailyFailed = &Error{Code: 5001114, Message: "读取每日统计失败。刷新页面后重试。", I18nKey: "apiError.5001114"}
	// ErrStatsCasesTopFailed — HTTP 500
	ErrStatsCasesTopFailed = &Error{Code: 5001115, Message: "读取用例排行失败。刷新页面后重试。", I18nKey: "apiError.5001115"}
	// ErrStatsErrorsFailed — HTTP 500
	ErrStatsErrorsFailed = &Error{Code: 5001116, Message: "读取错误统计失败。刷新页面后重试。", I18nKey: "apiError.5001116"}
	// ErrStatsFleetFailed — HTTP 500
	ErrStatsFleetFailed = &Error{Code: 5001117, Message: "读取集群统计失败。刷新页面后重试。", I18nKey: "apiError.5001117"}
	// ErrStudioUnavailable — HTTP 500
	ErrStudioUnavailable = &Error{Code: 5001205, Message: "工作台服务不可用。重启 `pixoma` 后重试。", I18nKey: "apiError.5001205"}
	// ErrStudioStreamAGUIFailed — HTTP 500
	ErrStudioStreamAGUIFailed = &Error{Code: 5001206, Message: "当前连接不支持流式响应。更换为支持 SSE 的环境。", I18nKey: "apiError.5001206"}
	// ErrMediaUploadFailed — HTTP 500
	ErrMediaUploadFailed = &Error{Code: 5001306, Message: "保存媒体失败。重新上传一次。", I18nKey: "apiError.5001306"}
	// ErrLinkHealthGetNotConfigured — HTTP 500
	ErrLinkHealthGetNotConfigured = &Error{Code: 5001405, Message: "链路健康服务未配置。检查配置后重启 `pixoma`。", I18nKey: "apiError.5001405"}
	// ErrLinkHealthGetFailed — HTTP 500
	ErrLinkHealthGetFailed = &Error{Code: 5001406, Message: "读取链路健康失败。刷新页面后重试。", I18nKey: "apiError.5001406"}
	// ErrAgentStatusNotConfigured — HTTP 500
	ErrAgentStatusNotConfigured = &Error{Code: 5001505, Message: "任务状态服务未配置。检查配置后重启 `pixoma`。", I18nKey: "apiError.5001505"}
	// ErrAgentPresenceFailed — HTTP 500
	ErrAgentPresenceFailed = &Error{Code: 5001506, Message: "上报在线状态失败。稍后重试。", I18nKey: "apiError.5001506"}
	// ErrAgentPresenceNotConfigured — HTTP 500
	ErrAgentPresenceNotConfigured = &Error{Code: 5001516, Message: "在线状态服务未配置。检查 `presence` 配置后重启 `pixoma`。", I18nKey: "apiError.5001516"}
	// ErrAgentHeartbeatFailed — HTTP 500
	ErrAgentHeartbeatFailed = &Error{Code: 5001517, Message: "上报心跳失败。稍后重试。", I18nKey: "apiError.5001517"}
	// ErrAgentClaimFailed — HTTP 500
	ErrAgentClaimFailed = &Error{Code: 5001518, Message: "领取任务失败。稍后重试。", I18nKey: "apiError.5001518"}

	// ErrStudioModelConnectionTest — HTTP 502
	ErrStudioModelConnectionTest = &Error{Code: 5021219, Message: "模型连接测试失败。检查 Base URL 和 API Key 后重新测试。", I18nKey: "apiError.5021219"}

	// ErrChannelSaveUnavailable — HTTP 503
	ErrChannelSaveUnavailable = &Error{Code: 5030815, Message: "模板存储未配置。检查 `channels` 配置后重启 `pixoma`。", I18nKey: "apiError.5030815"}
	// ErrStudioCreateModelUnavailable — HTTP 503
	ErrStudioCreateModelUnavailable = &Error{Code: 5031216, Message: "模型配置服务不可用。检查模型配置后重启 `pixoma`。", I18nKey: "apiError.5031216"}
	// ErrStudioListSkillsUnavailable — HTTP 503
	ErrStudioListSkillsUnavailable = &Error{Code: 5031217, Message: "能力配置服务不可用。检查能力配置后重启 `pixoma`。", I18nKey: "apiError.5031217"}
	// ErrStudioCreateTextAssetUnavailable — HTTP 503
	ErrStudioCreateTextAssetUnavailable = &Error{Code: 5031218, Message: "资产存储不可用。检查对象存储配置后重启 `pixoma`。", I18nKey: "apiError.5031218"}
)

// registry 按码索引全部错误，供测试与前端词条生成使用。
var registry = map[int]*Error{
	4000102: ErrSetupTestDatabaseDSNRequired,
	4000110: ErrSetupFinalizeDefaultPasswordUnchanged,
	4000111: ErrSetupSaveDraftInvalid,
	4000112: ErrSetupSaveSettingsInvalid,
	4000113: ErrSetupSaveSettingsSetupNotFinalized,
	4000114: ErrSetupFinalizeInvalid,
	4000115: ErrSetupChangePasswordPasswordAlreadySet,
	4000116: ErrSetupChangePasswordPasswordTooShort,
	4000117: ErrSetupTestBlobCheckFailed,
	4000118: ErrSetupTestDatabasePingFailed,
	4000119: ErrSetupSaveDraftDatabaseNotConfigured,
	4000120: ErrSetupTestDatabaseUnreachable,
	4000122: ErrSetupTestBlobConfigInvalid,
	4000123: ErrSetupTestDatabaseOpenFailed,
	4000125: ErrSetupSaveSettingsSettingsNotSaved,
	4000126: ErrSetupLoginInvalidJSON,
	4000127: ErrSetupLoadSettingsInvalid,
	4000128: ErrSetupRegisterAccountNameRequired,
	4000129: ErrSetupSaveProfileInvalidEmail,
	4000202: ErrUserUpdateAccessInvalidAccess,
	4000211: ErrUserUpdateAccessInvalidJSON,
	4000212: ErrUserListInvalidQuery,
	4000302: ErrAdminUserListInvalidLimit,
	4000310: ErrAdminUserCreatePasswordTooShort,
	4000311: ErrAdminUserUpdateInvalidRole,
	4000312: ErrAdminUserCreateInvalidJSON,
	4000313: ErrAdminUserCreateAccountNameRequired,
	4000314: ErrAdminUserCreateInvalidEmail,
	4000402: ErrSessionListInvalidQuery,
	4000502: ErrTaskListInvalidQuery,
	4000602: ErrCaseCreateInvalid,
	4000610: ErrCaseUpdateInvalid,
	4000611: ErrCaseGetInvalidCaseID,
	4000612: ErrCaseCreateInvalidJSON,
	4000613: ErrCaseListInvalidQuery,
	4000702: ErrEdgeListTasksInvalidLimit,
	4000710: ErrEdgeListTasksInvalidOffset,
	4000711: ErrEdgeCreateNameRequired,
	4000712: ErrEdgeUpdateTopicUnknown,
	4000713: ErrEdgeCreateInvalidJSON,
	4000714: ErrEdgeMetricsInvalidQuery,
	4000802: ErrChannelCreateInvalid,
	4000811: ErrChannelCreateMCPUserNameRequired,
	4000812: ErrChannelPutMenuInvalid,
	4000813: ErrChannelCreateMCPUserNotMCP,
	4000814: ErrChannelCreateInvalidJSON,
	4000902: ErrTopicCreateNameRequired,
	4000910: ErrTopicCreateInvalid,
	4000911: ErrTopicCreateInvalidJSON,
	4001102: ErrStatsErrorsInvalidLimit,
	4001110: ErrStatsCasesTopInvalidLimit,
	4001111: ErrStatsRangeReversed,
	4001112: ErrStatsInvalidRange,
	4001113: ErrStatsRangeTooLong,
	4001202: ErrStudioStreamAGUIMissingFields,
	4001212: ErrStudioUploadAssetFileMissing,
	4001213: ErrStudioStreamAGUIInvalidJSON,
	4001214: ErrStudioInvalidBody,
	4001215: ErrStudioUploadAssetFileReadFailed,
	4001302: ErrMediaUploadFileMissing,
	4001310: ErrMediaUploadFileTypeUnsupported,
	4001312: ErrMediaUploadInvalidContentType,
	4001313: ErrMediaUploadFileReadFailed,
	4001502: ErrAgentClaimEdgeIDRequired,
	4001510: ErrAgentPresenceInvalidJSON,
	4001511: ErrAgentHeartbeatInvalidJSON,
	4010007: ErrCommonPermissionUnauthorized,
	4010107: ErrSetupSessionUnauthorized,
	4010140: ErrSetupLoginUnauthorized,
	4011207: ErrStudioAccountUnauthorized,
	4011507: ErrAgentClaimUnauthorized,
	4030008: ErrCommonPermissionForbidden,
	4030108: ErrSetupNotInitialized,
	4030141: ErrSetupSessionForbidden,
	4030142: ErrSetupRestartRequired,
	4030143: ErrSetupAccountDisabled,
	4030208: ErrUserRotateMCPTokenForbidden,
	4040101: ErrSetupBlobBucketMissing,
	4040201: ErrUserLoadMCPTokenNotFound,
	4040210: ErrUserGetNotFound,
	4040301: ErrAdminUserNotFound,
	4040401: ErrSessionGetNotFound,
	4040501: ErrTaskGetNotFound,
	4040601: ErrCaseGetNotFound,
	4040701: ErrEdgeGetNotFound,
	4040801: ErrChannelGetNotFound,
	4040901: ErrTopicGetNotFound,
	4041201: ErrStudioNotFound,
	4041210: ErrStudioAssetNotFound,
	4041211: ErrStudioAssetVersionNotFound,
	4041301: ErrMediaServePreviewNotFound,
	4081512: ErrAgentClaimCanceled,
	4090103: ErrSetupRegisterRegistrationDisabled,
	4090130: ErrSetupRegisterAlreadyTaken,
	4090303: ErrAdminUserUpdateLastAdmin,
	4090315: ErrAdminUserDeleteLastAdmin,
	4090316: ErrAdminUserAlreadyTaken,
	4090503: ErrTaskCancelCancelNotAllowed,
	4090603: ErrCaseCreateAlreadyExists,
	4090604: ErrCaseDeleteConflict,
	4090703: ErrEdgeCreateAlreadyExists,
	4090704: ErrEdgeDeleteConflict,
	4090903: ErrTopicCreateAlreadyExists,
	4090904: ErrTopicDeleteConflict,
	4090912: ErrTopicUpdateDefaultProtected,
	4090913: ErrTopicDeleteDefaultProtected,
	4091203: ErrStudioAlreadyExists,
	4091503: ErrAgentStatusConflict,
	4091513: ErrAgentStatusTaskNotFound,
	4091514: ErrAgentStatusStaleHolder,
	4091515: ErrAgentHeartbeatHeartbeatRejected,
	4100810: ErrChannelGoneCardsUnknown,
	4130002: ErrCommonRequestBodyTooLarge,
	4131311: ErrMediaUploadTooLarge,
	4290121: ErrSetupRegisterTooManyRequests,
	4290124: ErrSetupLoginTooManyRequests,
	5000005: ErrCommonInternal,
	5000105: ErrSetupRegisterServiceUnavailable,
	5000106: ErrSetupSaveDraftFailed,
	5000131: ErrSetupSaveSettingsFailed,
	5000132: ErrSetupSaveProfileFailed,
	5000133: ErrSetupChangePasswordFailed,
	5000134: ErrSetupRegisterCreateFailed,
	5000135: ErrSetupFinalizeFailed,
	5000136: ErrSetupRegisterHashPasswordFailed,
	5000137: ErrSetupRegisterFailed,
	5000138: ErrSetupTestDatabaseFailed,
	5000139: ErrSetupLoginFailed,
	5000206: ErrUserLoadMCPTokenNotConfigured,
	5000213: ErrUserUpdateAccessFailed,
	5000214: ErrUserGetMCPTokenFailed,
	5000215: ErrUserListFailed,
	5000216: ErrUserRotateMCPTokenFailed,
	5000306: ErrAdminUserCreateFailed,
	5000317: ErrAdminUserDeleteDeleteFailed,
	5000318: ErrAdminUserCreateHashPasswordFailed,
	5000319: ErrAdminUserUpdateFailed,
	5000320: ErrAdminUserUpdateCountAdminsFailed,
	5000321: ErrAdminUserListListFailed,
	5000322: ErrAdminUserLoadFailed,
	5000406: ErrSessionListFailed,
	5000505: ErrTaskCancelNotConfigured,
	5000506: ErrTaskCancelFailed,
	5000510: ErrTaskListFailed,
	5000605: ErrCaseDeleteNotConfigured,
	5000606: ErrCaseDisableFailed,
	5000614: ErrCaseCreateFailed,
	5000615: ErrCaseDeleteFailed,
	5000616: ErrCaseEnableFailed,
	5000617: ErrCaseUpdateFailed,
	5000618: ErrCaseListFailed,
	5000705: ErrEdgeDeleteNotConfigured,
	5000706: ErrEdgeWriteFailed,
	5000715: ErrEdgeMetricsNotConfigured,
	5000716: ErrEdgeUpdateNotConfigured,
	5000717: ErrEdgeCreateFailed,
	5000718: ErrEdgeDeleteFailed,
	5000719: ErrEdgeUpdateFailed,
	5000720: ErrEdgeListTasksFailed,
	5000721: ErrEdgeListPresenceFailed,
	5000722: ErrEdgeListFailed,
	5000723: ErrEdgeMetricsFailed,
	5000724: ErrEdgeStatsFailed,
	5000725: ErrEdgeRotateTokenFailed,
	5000805: ErrChannelCreateMCPUserNotConfigured,
	5000806: ErrChannelDisableFailed,
	5000816: ErrChannelCreateNotConfigured,
	5000817: ErrChannelCreateFailed,
	5000818: ErrChannelDeleteMCPUserFailed,
	5000819: ErrChannelDeleteFailed,
	5000820: ErrChannelEnableFailed,
	5000821: ErrChannelGetMenuFailed,
	5000822: ErrChannelUpdateFailed,
	5000823: ErrChannelCreateMCPUserFailed,
	5000824: ErrChannelListMCPUsersFailed,
	5000825: ErrChannelListFailed,
	5000905: ErrTopicDeleteNotConfigured,
	5000906: ErrTopicCreateFailed,
	5000914: ErrTopicDeleteFailed,
	5000915: ErrTopicUpdateFailed,
	5000916: ErrTopicStatsFailed,
	5000917: ErrTopicListFailed,
	5001005: ErrRoutingAttributesNotConfigured,
	5001105: ErrStatsFleetNotConfigured,
	5001106: ErrStatsEdgesFailed,
	5001114: ErrStatsDailyFailed,
	5001115: ErrStatsCasesTopFailed,
	5001116: ErrStatsErrorsFailed,
	5001117: ErrStatsFleetFailed,
	5001205: ErrStudioUnavailable,
	5001206: ErrStudioStreamAGUIFailed,
	5001306: ErrMediaUploadFailed,
	5001405: ErrLinkHealthGetNotConfigured,
	5001406: ErrLinkHealthGetFailed,
	5001505: ErrAgentStatusNotConfigured,
	5001506: ErrAgentPresenceFailed,
	5001516: ErrAgentPresenceNotConfigured,
	5001517: ErrAgentHeartbeatFailed,
	5001518: ErrAgentClaimFailed,
	5021219: ErrStudioModelConnectionTest,
	5030815: ErrChannelSaveUnavailable,
	5031216: ErrStudioCreateModelUnavailable,
	5031217: ErrStudioListSkillsUnavailable,
	5031218: ErrStudioCreateTextAssetUnavailable,
}

// ByCode 按码查错误。
func ByCode(code int) (*Error, bool) {
	e, ok := registry[code]
	return e, ok
}

// All 返回全部错误，按码升序。
func All() []*Error {
	out := make([]*Error, 0, len(registry))
	for _, e := range registry {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}
