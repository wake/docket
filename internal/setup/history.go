package setup

// Earlier maps a target path to the SHA-256 (lowercase hex) of the files
// earlier releases shipped there; the key for the operations section of
// openspec/config.yaml is "openspec/config.yaml#operations". A change to a
// shipped file adds its new hash here, and TestEarlierHoldsShippedFiles fails
// until it does. Tests in other packages may replace it, and must restore it.
var Earlier = map[string][]string{
	".agents/skills/docket/SKILL.md": {
		"0930d8fc316c0c7a152cbf3535de013bbc5e5ac9918f97444099f27e051b5a18", // v0.16.0
		"d2d5213bb1fca7d3856b2e2bcaf334e43d28d1f3ca8ca250c68326fea922f193", // v0.17.0
		"8552f47bef60b0ca511c9cd2eb29a50e19732085203581e881caf0084d797490", // v0.18.0
	},
	".claude/skills/docket/SKILL.md": {
		"0930d8fc316c0c7a152cbf3535de013bbc5e5ac9918f97444099f27e051b5a18", // v0.16.0
		"d2d5213bb1fca7d3856b2e2bcaf334e43d28d1f3ca8ca250c68326fea922f193", // v0.17.0
		"8552f47bef60b0ca511c9cd2eb29a50e19732085203581e881caf0084d797490", // v0.18.0
	},
	"openspec/config.yaml#operations": {
		"e3f455c4cdf31ed8f6b754df7e912832d60e3583d5376cfb0afd1ad7b22c5753", // v0.16.0, v0.17.0, v0.18.0
	},
	"openspec/schemas/docket/schema.yaml": {
		"5aeaf2bbbbf4ee8f2231f2f7d16d0de079ebe7f1257a3d5381d77467171e4550", // v0.16.0, v0.17.0, v0.18.0
	},
	"openspec/schemas/docket/templates/design.md": {
		"6ccc87f5017fb190e69e8759dd4e20d5ac3320cc4dabf152708bf8f2aa5205f8", // v0.16.0, v0.17.0, v0.18.0
	},
	"openspec/schemas/docket/templates/plan.md": {
		"9f58b30fa65f74b3893423d73b94f64d04f37cb18db575ab43e0df4d1ea545a4", // v0.16.0, v0.17.0, v0.18.0
	},
	"openspec/schemas/docket/templates/proposal.md": {
		"baa96c14f1c7540b6ea644628ec88e192832e92a7290c6b40bc40dffb8344d31", // v0.16.0, v0.17.0, v0.18.0
	},
	"openspec/schemas/docket/templates/spec.md": {
		"1cccfb573f031c6352bed2a879b7035ff710bc909112fc23d10bdcfcdd8d4318", // v0.16.0, v0.17.0, v0.18.0
	},
}
