package includes

// Incoming JSON document. Must be checked for below fields.
type InitSession struct {
	OrangeTeamName *string  `json:"orangeTeamName"`
	BlueTeamName   *string  `json:"blueTeamName"`
	MapPool        []string `json:"mapPool"`
}

// Session represents a mapban session. This implements the node
// of a disjoint set.
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

	// Map pool, all maps chosen for the session
	MapPool []string `json:"mapPool"`

	// Follows the following format: [OT pick, BT pick, decider]
	MapsChosen []string `json:"mapsChosen"`

	// Tracks the current phase of the session
	CurrentPhase int `json:"-"`
}

type MapChoice struct {
	// Token
	Token *string `json:"token"`

	// Map chosen (used for both bans and picks)
	Choice *string `json:"choice"`
}
