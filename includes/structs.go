package includes

type TeamNames struct {
	OrangeTeamName string `json:"orangeTeamName"`
	BlueTeamName   string `json:"blueTeamName"`
}

// Session represents a mapban session.
// JSON corresponding format:
//
//	{
//		orangeTeamToken: "...",
//		blueTeamToken: "...",
//		orangeTeamName: "...",
//		blueTeamName: "...",
//		mapsChosen: ["...", "...", ...]
// 	}
//
type Session struct {
	// Host token
	HostToken string `json:"hostToken"`

	// Team tokens and names
	OrangeTeamToken string `json:"orangeTeamToken"`
	BlueTeamToken   string `json:"blueTeamToken"`
	OrangeTeamName  string `json:"orangeTeamName"`
	BlueTeamName    string `json:"blueTeamName"`

	// Follows the following format: [OT ban, BT ban, OT pick, BT pick, OT ban, BT ban, decider]
	MapsChosen []string `json:"mapsChosen"`
}
