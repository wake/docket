package setup

// Earlier maps a target path to the SHA-256 (lowercase hex) of the files
// earlier releases shipped there; the key for the operations section of
// openspec/config.yaml is "openspec/config.yaml#operations". A change to a
// shipped file adds its new hash here, and TestEarlierHoldsShippedFiles fails
// until it does; TestEarlierHoldsReleases fails when a hash of a release is
// gone. Tests in other packages may replace it, and must restore it.
var Earlier = map[string][]string{
	".agents/skills/docket/SKILL.md": {
		"0930d8fc316c0c7a152cbf3535de013bbc5e5ac9918f97444099f27e051b5a18", // v0.16.0
		"d2d5213bb1fca7d3856b2e2bcaf334e43d28d1f3ca8ca250c68326fea922f193", // v0.17.0
		"8552f47bef60b0ca511c9cd2eb29a50e19732085203581e881caf0084d797490", // v0.18.0
		"74b12750a28b2af14d7f2717ce35f2e237a30ca135c57bb86880562fbe605f7a", // v0.22.0
		"40503ca11812dae05ff3e2e2c4fdc5e7f13661bb0b8f0b1dc7dbd8e32eb614a8", // v0.24.2
		"ca265ef4ab31fcc0a8c96324bd30ab6091512431337c530f52e8fe1d04085e09", // v0.27.0
		"891c3b3f3ef767bc03158be092829d18922747506b1599563f1fea9dc1998027", // v0.28.0
	},
	".agents/skills/docket/domain-tree.md": {
		"13e8d75866c692b0bf5936b7a877a74d3026b3a4613018284fd14c368eec3d70", // v0.28.0
	},
	".agents/skills/docket/issues-in-another-tool.md": {
		"cf438e83ed3e6f7f0db63c31c82e28a695d6fe4f37197773c1f8e8147d66da7d", // v0.28.0
	},
	".claude/skills/docket/SKILL.md": {
		"0930d8fc316c0c7a152cbf3535de013bbc5e5ac9918f97444099f27e051b5a18", // v0.16.0
		"d2d5213bb1fca7d3856b2e2bcaf334e43d28d1f3ca8ca250c68326fea922f193", // v0.17.0
		"8552f47bef60b0ca511c9cd2eb29a50e19732085203581e881caf0084d797490", // v0.18.0
		"74b12750a28b2af14d7f2717ce35f2e237a30ca135c57bb86880562fbe605f7a", // v0.22.0
		"40503ca11812dae05ff3e2e2c4fdc5e7f13661bb0b8f0b1dc7dbd8e32eb614a8", // v0.24.2
		"ca265ef4ab31fcc0a8c96324bd30ab6091512431337c530f52e8fe1d04085e09", // v0.27.0
		"891c3b3f3ef767bc03158be092829d18922747506b1599563f1fea9dc1998027", // v0.28.0
	},
	".claude/skills/docket/domain-tree.md": {
		"13e8d75866c692b0bf5936b7a877a74d3026b3a4613018284fd14c368eec3d70", // v0.28.0
	},
	".claude/skills/docket/issues-in-another-tool.md": {
		"cf438e83ed3e6f7f0db63c31c82e28a695d6fe4f37197773c1f8e8147d66da7d", // v0.28.0
	},
	"openspec/config.yaml#operations": {
		"e3f455c4cdf31ed8f6b754df7e912832d60e3583d5376cfb0afd1ad7b22c5753", // v0.16.0, v0.17.0, v0.18.0
	},
	"openspec/schemas/docket/schema.yaml": {
		"5aeaf2bbbbf4ee8f2231f2f7d16d0de079ebe7f1257a3d5381d77467171e4550", // v0.16.0, v0.17.0, v0.18.0
		"b7126bff76ee3b91994dca48cc0a28f4b7fc38393776b5ea5498e73b2e530589", // v0.26.0
		"93357a05020f74fd4e847d038831513ea0a18b9687c39c5648ac5a6c2c2f617e", // v0.27.0
		"b97feb0a9a40cac937b55eb03daaca66ea9fb283c362f68db99dcadd46c0cb0a", // v0.28.0
		"508dd52f2098cca3d175f4a79cdd4193b6411d20562bfb2d364518411d0b2923", // v0.29.0
	},
	"openspec/schemas/docket/templates/design.md": {
		"6ccc87f5017fb190e69e8759dd4e20d5ac3320cc4dabf152708bf8f2aa5205f8", // v0.16.0, v0.17.0, v0.18.0
	},
	"openspec/schemas/docket/templates/plan.md": {
		"9f58b30fa65f74b3893423d73b94f64d04f37cb18db575ab43e0df4d1ea545a4", // v0.16.0, v0.17.0, v0.18.0
		"65d6b1cd124cad1888a8b6be96e6542aabc75d40ed072c1c3edf91978e58fb1d", // v0.26.0
		"cde45107190618d0a13258c5698c4af728961cee197eca112f549383bba0ebd9", // v0.27.0
	},
	"openspec/schemas/docket/templates/proposal.md": {
		"baa96c14f1c7540b6ea644628ec88e192832e92a7290c6b40bc40dffb8344d31", // v0.16.0, v0.17.0, v0.18.0
		"260501b1b54e63727c45e5d5979347921f8dc4fe67a982ad8d4aa9677f96627e", // v0.22.0
	},
	"openspec/schemas/docket/templates/spec.md": {
		"1cccfb573f031c6352bed2a879b7035ff710bc909112fc23d10bdcfcdd8d4318", // v0.16.0, v0.17.0, v0.18.0
	},
}
