package location

// Sample UN locodes.
var (
	SESTO UNLcode = "SESTO"
	AUMEL UNLcode = "AUMEL"
	CNHKG UNLcode = "CNHKG"
	USNYC UNLcode = "USNYC"
	USCHI UNLcode = "USCHI"
	JNTKO UNLcode = "JNTKO"
	DEHAM UNLcode = "DEHAM"
	NLRTM UNLcode = "NLRTM"
	FIHEL UNLcode = "FIHEL"
)

// Sample locations.
var (
	Stockholm = &Location{SESTO, "Stockholm"}
	Melbourne = &Location{AUMEL, "Melbourne"}
	Hongkong  = &Location{CNHKG, "Hongkong"}
	NewYork   = &Location{USNYC, "New York"}
	Chicago   = &Location{USCHI, "Chicago"}
	Tokyo     = &Location{JNTKO, "Tokyo"}
	Hamburg   = &Location{DEHAM, "Hamburg"}
	Rotterdam = &Location{NLRTM, "Rotterdam"}
	Helsinki  = &Location{FIHEL, "Helsinki"}
)