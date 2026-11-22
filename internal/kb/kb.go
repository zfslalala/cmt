// Package kb 把一个仓库之外的知识库挂载到 git 工作树。
//
// 知识库磁盘布局：
//
//	<知识库>/projects/<项目>/base/              基座：已确认的知识
//	<知识库>/projects/<项目>/branches/<分支>/   分支自己的知识，可写、与基座隔离
//
// 主工作树直接指向基座，基座就是在主工作树上编辑的。特性分支挂载时在工作树里
// 生成 .dsh 合成视图：技能按名字由分支覆盖基座，knowledge 指向分支目录，
// base 指向基座作为只读参考。同一份技能入口同时暴露给 Codex (.agents) 与
// Qoder (.qoder)，工作树根 AGENTS.md 则合并基座与 AGENTS.local.md。
// 所有挂载物都写进 git 的 info/exclude，因此对 git 不可见，也不会被提交。
package kb

const (
	// DshDir 是工作树里承载技能与知识的目录名。
	DshDir = ".dsh"
	// LocalNoteFile 是本工作树专属的说明文件，会被合并进工作树根 AGENTS.md。
	LocalNoteFile = "AGENTS.local.md"
	// defaultRootName 是知识库的默认目录名，与仓库同级。
	defaultRootName = "_ai-kb"
	// baseDirName 是基座目录名。
	baseDirName = "base"
	// branchesDirName 是分支知识目录名。
	branchesDirName = "branches"
	// projectKeyFile 是仓库本地覆盖项目键的文件，位于共享 git 目录的 info 下。
	projectKeyFile = "kb-project"
	// managedMarker 标记 .dsh 目录由 qg 创建，可安全重建。
	managedMarker = ".qg-managed"
	// instructionsMarker 标记工作树根 AGENTS.md 由 qg 从知识库生成。
	instructionsMarker = "<!-- qg-managed: agent-instructions -->"
	// SharedAgentsDir 是 Codex 等项目级 agent 共享技能目录。
	SharedAgentsDir = ".agents"
	// QoderDir 是 Qoder 的项目规则与技能目录。
	QoderDir = ".qoder"
)

// excludeEntries 会被写进 git 的 info/exclude，让挂载物对 git 不可见。
var excludeEntries = []string{
	"/AGENTS.md",
	"/AGENTS.local.md",
	"/CLAUDE.md",
	"/GEMINI.md",
	"/.dsh",
	"/.agents",
	"/.qoder",
	"/.trae",
}
