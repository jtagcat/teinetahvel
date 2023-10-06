package tahvel

const baseURL = "https://tahvel.edu.ee"

var roomAclMapping = map[string]string{
	"D107": "harjutus",
	"D108": "harjutus",
	"D109": "harjutus",
	"D115": "harjutus",
	"D118": "harjutus",
	"D301": "harjutus",
	"D305": "harjutus",
	"D307": "harjutus",
	"D308": "harjutus",
	"D313": "harjutus",
	"D314": "harjutus",
	"D315": "harjutus",
	"D325": "harjutus",
}

var aclMapping = map[string][]string{
	"KJ": {"üld", "klassika", "harjutus"}, // Koorijuhtimine
	// "AK":      {},                              // Akordion?
	// "KA":      {},                              // Kannel?
	"KL": {"üld", "klassika", "harjutus"}, // Klahvpillid
	// "OR":      {},
	// "KH":      {}, // Klaverihäälestaja
	// "KO":      {},
	// "MT":      {},
	// "KP":      {},
	// "KI":      {},
	// "LA":      {},                                          // (Klassikaline?) laul
	// "PP":      {},                                          // Puhkpill
	"LP": {"üld", "klassika", "harjutus", "löökpill"}, // Klassikaline löökpill
	// "RP":      {},                                          // Rütmimuusika puhkpill
	// "RL":      {},                                          // Rütmimuusika löökpill
	// "MM":      {},                                          // Maailmamuusik
	// "HE":      {},                                          // Helindaja
	// "PJ":      {},                                          // Indrek Koff, juhataja? Puhkpill Pillijuhendaja?
	// "RM(MP)-": {},
	// "ME":      {},
}

// https://muba.edu.ee/wp-content/uploads/2023/02/muba-broneeritavad-klaverid-6-02-2023-2.pdf
var pianoRooms = map[string]int{
	// Kahe klaveriga klassid
	"D303": 2,
	"D306": 2,
	"D309": 2,
	"D312": 2,
	"D316": 2,
	"D319": 2,
	"D321": 2,
	"D324": 2,
	"D326": 2,
	"C107": 2,
	"C108": 2,
	"C110": 2,
	"B401": 2,
	"B402": 2,
	"B404": 2,
	"B405": 2,
	"B406": 2,
	"B407": 2,
	"B408": 2,
	"B409": 2,
	"B410": 2,
	"B411": 2,
	"B412": 2,
	"B413": 2,
	"B414": 2,
	"B416": 2,
	"B417": 2,

	// Harjutusruumid
	"D107": 1,
	"D108": 1,
	"D109": 1,
	"D115": 1,
	"D118": 1,
	"D301": 1,
	"D305": 1,
	"D307": 1,
	"D308": 1,
	"D313": 1,
	"D314": 1,
	"D315": 1,
	"D325": 1,

	// Üldklaver + teised pillid
	"C106": 1,
	"C109": 1,
	"C111": 1,

	// Lauluklassid
	"C112": 1,
	"C113": 1,
	"C114": 1,

	// Ansambliklassid
	"C115": 1,
	"C116": 1,

	// Muud klassid
	"A212": 2,
	"B104": 1,

	// Keelpilliklassid
	"B302": 1,
	"B304": 1,
	"B305": 1,
	"B306": 1,
	"B307": 1,
	"B308": 1,
	"B309": 1,
	"B310": 1,
	"B311": 1,
	"B312": 1,
	"B313": 1,
	"B314": 1,
	"B315": 1,
	"B317": 1,
	"B303": 1,

	// Puhkpillide klassid
	"B201": 1,
	"B202": 1,
	"B205": 1,
	"B207": 1,
	"B208": 1,
	"B209": 1,
	"B211": 1,
	"B204": 1,
	"B206": 1,
	"B203": 1,

	// Esinejate ruumid
	"A109": 1,
	"A121": 1,

	// Lisaklassid
	"A213": 2,
	"D112": 1,
	"B301": 1,
	"B403": 1,
	"B102": 1,
	"C103": 1,

	// Üldainete klassid
	"C310": 1,
	"C308": 1,
	"C312": 1,
	"C309": 1,
	"C311": 1,
	"C306": 1,
	"C408": 1,
	"C410": 1,
	"C411": 1,
	"C206": 1,
}
