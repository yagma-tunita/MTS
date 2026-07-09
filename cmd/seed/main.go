package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"backend/internal/model"
	"backend/pkg/config"
	"backend/pkg/crypto"
	"backend/pkg/database"

	"gorm.io/gorm"
)

func main() {
	cfg := config.MustLoad("config.yaml")
	fmt.Println("Connecting to database...")
	db := database.MustNewMySQL(cfg.Database, "info", 200*time.Millisecond)
	fmt.Println("Connected, starting seed...")
	cleanDB(db)
	seed(db)
	fmt.Println("Seed completed!")
}

func cleanDB(db *gorm.DB) {
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	tables := []string{
		"segment_capacity_usage", "order_cargo", "shipping_order",
		"voyage_cargo_note", "voyage_berthing", "shipping_line",
		"vessel", "berth", "port", "city",
		"shipper_company", "shipping_company", "admin", "cargo_type",
		"line_vessel",
	}
	for _, t := range tables {
		db.Exec("DELETE FROM " + t)
		db.Exec("ALTER TABLE " + t + " AUTO_INCREMENT = 1")
	}
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	fmt.Println("Cleaned all tables")
}

func seed(db *gorm.DB) {
	now := time.Now()
	rng := rand.New(rand.NewSource(42))

	// ===== 20 Cities =====
	cities := []model.City{
		{CityName: "Guangzhou", Country: strPtr("China"), CountryCode: strPtr("CN"), Timezone: strPtr("Asia/Shanghai")},
		{CityName: "Shanghai", Country: strPtr("China"), CountryCode: strPtr("CN"), Timezone: strPtr("Asia/Shanghai")},
		{CityName: "Hong Kong", Country: strPtr("China"), CountryCode: strPtr("CN"), Timezone: strPtr("Asia/Hong_Kong")},
		{CityName: "Shenzhen", Country: strPtr("China"), CountryCode: strPtr("CN"), Timezone: strPtr("Asia/Shanghai")},
		{CityName: "Ningbo", Country: strPtr("China"), CountryCode: strPtr("CN"), Timezone: strPtr("Asia/Shanghai")},
		{CityName: "Singapore", Country: strPtr("Singapore"), CountryCode: strPtr("SG"), Timezone: strPtr("Asia/Singapore")},
		{CityName: "Mumbai", Country: strPtr("India"), CountryCode: strPtr("IN"), Timezone: strPtr("Asia/Kolkata")},
		{CityName: "Colombo", Country: strPtr("Sri Lanka"), CountryCode: strPtr("LK"), Timezone: strPtr("Asia/Colombo")},
		{CityName: "Durban", Country: strPtr("South Africa"), CountryCode: strPtr("ZA"), Timezone: strPtr("Africa/Johannesburg")},
		{CityName: "Cape Town", Country: strPtr("South Africa"), CountryCode: strPtr("ZA"), Timezone: strPtr("Africa/Johannesburg")},
		{CityName: "Rotterdam", Country: strPtr("Netherlands"), CountryCode: strPtr("NL"), Timezone: strPtr("Europe/Amsterdam")},
		{CityName: "Hamburg", Country: strPtr("Germany"), CountryCode: strPtr("DE"), Timezone: strPtr("Europe/Berlin")},
		{CityName: "Santos", Country: strPtr("Brazil"), CountryCode: strPtr("BR"), Timezone: strPtr("America/Sao_Paulo")},
		{CityName: "Rio de Janeiro", Country: strPtr("Brazil"), CountryCode: strPtr("BR"), Timezone: strPtr("America/Sao_Paulo")},
		{CityName: "Tokyo", Country: strPtr("Japan"), CountryCode: strPtr("JP"), Timezone: strPtr("Asia/Tokyo")},
		{CityName: "Busan", Country: strPtr("South Korea"), CountryCode: strPtr("KR"), Timezone: strPtr("Asia/Seoul")},
		{CityName: "Dubai", Country: strPtr("UAE"), CountryCode: strPtr("AE"), Timezone: strPtr("Asia/Dubai")},
		{CityName: "Piraeus", Country: strPtr("Greece"), CountryCode: strPtr("GR"), Timezone: strPtr("Europe/Athens")},
		{CityName: "Los Angeles", Country: strPtr("USA"), CountryCode: strPtr("US"), Timezone: strPtr("America/Los_Angeles")},
		{CityName: "Long Beach", Country: strPtr("USA"), CountryCode: strPtr("US"), Timezone: strPtr("America/Los_Angeles")},
	}
	for i := range cities { cities[i].CreateTime = now; cities[i].UpdateTime = now; mustCreate(db, &cities[i]) }
	fmt.Println("Created 20 cities")

	// ===== 20 Ports =====
	ports := []model.Port{
		{PortName: "Guangzhou Nansha Port", PortCode: strPtr("CNNSH"), CityID: &cities[0].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(15.0)},
		{PortName: "Shanghai Yangshan Port", PortCode: strPtr("CNSHA"), CityID: &cities[1].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(17.0)},
		{PortName: "Hong Kong Port", PortCode: strPtr("HKHKG"), CityID: &cities[2].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(16.5)},
		{PortName: "Shenzhen Yantian Port", PortCode: strPtr("CNYTN"), CityID: &cities[3].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(16.0)},
		{PortName: "Ningbo Zhoushan Port", PortCode: strPtr("CNNGB"), CityID: &cities[4].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(16.5)},
		{PortName: "Singapore Port", PortCode: strPtr("SGSIN"), CityID: &cities[5].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(17.0)},
		{PortName: "Mumbai Port", PortCode: strPtr("INBOM"), CityID: &cities[6].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(14.0)},
		{PortName: "Colombo Port", PortCode: strPtr("LKCMB"), CityID: &cities[7].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(15.0)},
		{PortName: "Durban Port", PortCode: strPtr("ZADUR"), CityID: &cities[8].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(15.0)},
		{PortName: "Cape Town Port", PortCode: strPtr("ZACPT"), CityID: &cities[9].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(16.0)},
		{PortName: "Rotterdam Port", PortCode: strPtr("NLRTM"), CityID: &cities[10].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(17.0)},
		{PortName: "Hamburg Port", PortCode: strPtr("DEHAM"), CityID: &cities[11].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(16.5)},
		{PortName: "Santos Port", PortCode: strPtr("BRSSZ"), CityID: &cities[12].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(16.0)},
		{PortName: "Rio de Janeiro Port", PortCode: strPtr("BRRIO"), CityID: &cities[13].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(15.5)},
		{PortName: "Tokyo Port", PortCode: strPtr("JPTYO"), CityID: &cities[14].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(15.5)},
		{PortName: "Busan Port", PortCode: strPtr("KRPUS"), CityID: &cities[15].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(16.0)},
		{PortName: "Dubai Port", PortCode: strPtr("AEDXB"), CityID: &cities[16].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(15.0)},
		{PortName: "Piraeus Port", PortCode: strPtr("GRPIR"), CityID: &cities[17].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(16.5)},
		{PortName: "Los Angeles Port", PortCode: strPtr("USLAX"), CityID: &cities[18].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(16.0)},
		{PortName: "Long Beach Port", PortCode: strPtr("USLGB"), CityID: &cities[19].CityID, PortType: strPtr("Sea Port"), MaxDraftMeter: f64Ptr(16.0)},
	}
	for i := range ports { ports[i].CreateTime = now; ports[i].UpdateTime = now; mustCreate(db, &ports[i]) }
	fmt.Println("Created 20 ports")

	// ===== Berths =====
	for i := 0; i < 20; i++ {
		for j := 0; j < 2; j++ {
			b := model.Berth{
				BerthName: fmt.Sprintf("%s Berth %d", ports[i].PortName, j+1), PortID: &ports[i].PortID,
				BerthType: strPtr([]string{"Container", "Bulk"}[j]), DraftMeter: f64Ptr(12.0 + float64(j)*2.5),
				LengthMeter: f64Ptr(280 + float64(j*50)), WidthMeter: f64Ptr(40 + float64(j*10)),
				MaxBerthingTonnage: f64Ptr(60000 + float64(j*20000)), FunctionalZone: strPtr(fmt.Sprintf("Zone %c", 'A'+j)),
				IsAvailable: 1, CreateTime: now, UpdateTime: now,
			}
			mustCreate(db, &b)
		}
	}
	fmt.Println("Created 40 berths")

	// ===== Shipper Companies =====
	shipperData := []struct{ name, credit, rep, phone, addr, user string }{
		{"中国远洋贸易有限公司", "91440101MA5XXXXXX1", "张伟", "13800138001", "广州天河区黄埔大道西100号", "shipper01"},
		{"上海国际矿业资源公司", "91310000MA5XXXXXX2", "李强", "13800138002", "上海浦东新区陆家嘴金融区100号", "shipper02"},
		{"深圳华远物流有限公司", "", "王明", "13800138006", "深圳南山区蛇口工业区", "shipper03"},
		{"北京东方供应链公司", "", "刘洋", "13800138008", "北京朝阳区国贸大厦", "shipper04"},
		{"天津港保税区贸易公司", "", "陈静", "13800138009", "天津滨海新区", "shipper05"},
	}
	shippers := make([]model.ShipperCompany, 0)
	for _, sd := range shipperData {
		sp, _ := crypto.HashPassword("123456")
		s := model.ShipperCompany{
			CompanyName: sd.name, UnifiedSocialCreditCode: strPtrOrNil(sd.credit),
			LegalRepresentative: strPtr(sd.rep), ContactPhone: strPtr(sd.phone),
			Address: strPtr(sd.addr), LoginUsername: sd.user, LoginPassword: sp,
			AccountStatus: 1, CreateTime: now, UpdateTime: now,
		}
		mustCreate(db, &s)
		shippers = append(shippers, s)
	}
	fmt.Printf("Created %d shippers\n", len(shippers))

	// ===== Shipping Companies =====
	type scData struct{ name, contact, phone, addr, user string }
	scList := []scData{
		{"中国远洋海运集团", "王建国", "13800138003", "北京朝阳区建国路100号", "cosco"},
		{"马士基航运(中国)有限公司", "Peter Jensen", "13800138004", "上海黄浦区南京西路100号", "maersk"},
		{"地中海航运有限公司", "陈志强", "13800138005", "深圳南山区蛇口工业区", "msc"},
		{"达飞轮船(中国)有限公司", "Jean Dupont", "13800138007", "上海静安区南京西路200号", "cma"},
	}
	companies := make([]*model.ShippingCompany, 0)
	for _, sc := range scList {
		p, _ := crypto.HashPassword("123456")
		c := &model.ShippingCompany{
			CompanyName: sc.name, ContactPerson: strPtr(sc.contact),
			ContactPhone: strPtr(sc.phone), Address: strPtr(sc.addr),
			LoginUsername: sc.user, LoginPassword: p, AccountStatus: 1,
			CreateTime: now, UpdateTime: now,
		}
		mustCreate(db, c)
		companies = append(companies, c)
	}
	fmt.Printf("Created %d shipping companies\n", len(companies))

	cosco, maersk, msc, cma := companies[0], companies[1], companies[2], companies[3]

	// ===== Admin =====
	ap, _ := crypto.HashPassword("admin123")
	mustCreate(db, &model.Admin{Username: "admin", Password: ap, RealName: strPtr("系统管理员"), Role: 1, CreateTime: now, UpdateTime: now})
	fmt.Println("Created admin (admin/admin123)")

	// ===== 24 Vessels (6 per company) =====
	type vData struct{ name, call, imo, vtype string; dwt, speed float64; teu int32 }
	vesselsData := map[int64][]vData{
		cosco.CompanyID: {
			{"远洋号 (Yuan Yang Hao)", "BOCQ", "IMO1000001", "Bulk Carrier", 300, 14.5, 3000},
			{"东方巨龙 (Oriental Dragon)", "BODR", "IMO1000002", "Container Ship", 120000, 22.0, 8000},
			{"丝路号 (Silk Road)", "BSLR", "IMO1000006", "Container Ship", 90000, 21.0, 6000},
			{"郑和号 (Zheng He)", "BZHE", "IMO1000007", "Container Ship", 150000, 24.0, 10000},
			{"长城号 (Great Wall)", "BGW", "IMO1000019", "Container Ship", 130000, 23.0, 9000},
			{"熊猫号 (Panda)", "BPND", "IMO1000020", "Bulk Carrier", 80000, 16.0, 2000},
		},
		maersk.CompanyID: {
			{"Maersk Guangzhou", "MGRZ", "IMO1000003", "Container Ship", 75000, 20.5, 5000},
			{"Maersk Shanghai", "MSH", "IMO1000009", "Container Ship", 180000, 23.5, 14000},
			{"Maersk Mumbai", "MMB", "IMO1000010", "Container Ship", 85000, 21.0, 6000},
			{"Maersk Copenhagen", "MCPH", "IMO1000013", "Container Ship", 95000, 22.0, 7000},
			{"Maersk Rotterdam", "MRDM", "IMO1000021", "Container Ship", 200000, 24.0, 16000},
			{"Maersk Singapore", "MSGP", "IMO1000022", "Container Ship", 110000, 21.5, 8000},
		},
		msc.CompanyID: {
			{"MSC Shanghai", "MSSH", "IMO1000005", "Container Ship", 180000, 23.0, 12000},
			{"MSC Tokyo", "MSTK", "IMO1000011", "Container Ship", 220000, 25.0, 20000},
			{"MSC Hamburg", "MSHB", "IMO1000012", "Oil Tanker", 200000, 16.5, 0},
			{"MSC Geneva", "MSG", "IMO1000014", "Container Ship", 130000, 22.5, 10000},
			{"MSC Paris", "MSPR", "IMO1000023", "Container Ship", 160000, 23.0, 12000},
			{"MSC London", "MSLD", "IMO1000024", "Container Ship", 190000, 24.0, 15000},
		},
		cma.CompanyID: {
			{"CMA CGM Paris", "CMAP", "IMO1000015", "Container Ship", 140000, 22.5, 11000},
			{"CMA CGM Marseille", "CMAM", "IMO1000016", "Container Ship", 110000, 21.5, 8000},
			{"CMA CGM Lyon", "CMAL", "IMO1000017", "Bulk Carrier", 60000, 15.5, 2000},
			{"CMA CGM Nice", "CMAN", "IMO1000018", "Container Ship", 170000, 23.0, 13000},
			{"CMA CGM Dubai", "CMDB", "IMO1000025", "Container Ship", 120000, 22.0, 9000},
			{"CMA CGM Tokyo", "CMTK", "IMO1000026", "Container Ship", 150000, 22.5, 11000},
		},
	}
	vessels := make([]model.Vessel, 0)
	for cid, vlist := range vesselsData {
		for _, v := range vlist {
			teu := v.teu
			ves := model.Vessel{
				VesselName: v.name, CallSign: strPtr(v.call), IMONumber: v.imo, VesselType: strPtr(v.vtype),
				MaxDeadweightTon: f64Ptr(v.dwt), GrossTonnage: f64Ptr(v.dwt * 0.75), SpeedKnot: f64Ptr(v.speed),
				DraftMeter: f64Ptr(12.0 + float64(v.teu)/5000*2), ContainerTEU: &teu, IsAvailable: 1,
				ShippingCompanyID: &cid, CreateTime: now, UpdateTime: now,
			}
			vessels = append(vessels, ves)
		}
	}
	for i := range vessels { mustCreate(db, &vessels[i]) }
	fmt.Printf("Created %d vessels (6 per company)\n", len(vessels))

	// vessel index ranges per company: cosco[0-5], maersk[6-11], msc[12-17], cma[18-23]
	vesselRange := map[int64][2]int{
		cosco.CompanyID:  {0, 5},
		maersk.CompanyID: {6, 11},
		msc.CompanyID:    {12, 17},
		cma.CompanyID:    {18, 23},
	}
	getCompanyVessels := func(cid int64) []int {
		r := vesselRange[cid]
		ids := make([]int, 0, r[1]-r[0]+1)
		for i := r[0]; i <= r[1]; i++ { ids = append(ids, i) }
		return ids
	}

	// ===== 15 Shipping Lines =====
	type lData struct{ name, depPort, destPort, desc, seq string; dist float64; cid int64 }
	linesData := []lData{
		{name: "中国-南美东海岸航线 (Guangzhou-Rio)", depPort: "Guangzhou Nansha Port", destPort: "Rio de Janeiro Port", desc: "经新加坡、孟买、德班、桑托斯至里约热内卢", seq: "[1,6,7,9,13,14]", dist: 12000, cid: cosco.CompanyID},
		{name: "亚欧航线 (Shanghai-Rotterdam via Suez)", depPort: "Shanghai Yangshan Port", destPort: "Rotterdam Port", desc: "亚洲至欧洲主力航线", seq: "[2,6,17,18,11]", dist: 8500, cid: cosco.CompanyID},
		{name: "东北亚-欧洲航线 (Busan-Rotterdam)", depPort: "Busan Port", destPort: "Rotterdam Port", desc: "连接韩日与欧洲", seq: "[16,15,2,6,11]", dist: 9200, cid: maersk.CompanyID},
		{name: "亚洲-非洲航线 (Hong Kong-Durban)", depPort: "Hong Kong Port", destPort: "Durban Port", desc: "经东南亚至南部非洲", seq: "[3,6,17,7,9]", dist: 7800, cid: msc.CompanyID},
		{name: "东北亚-地中海航线 (Tokyo-Piraeus)", depPort: "Tokyo Port", destPort: "Piraeus Port", desc: "连接日本至地中海", seq: "[15,16,2,6,18]", dist: 7500, cid: maersk.CompanyID},
		{name: "华南-中东航线 (Guangzhou-Mumbai)", depPort: "Guangzhou Nansha Port", destPort: "Mumbai Port", desc: "华南至印度", seq: "[1,3,6,17,7]", dist: 4500, cid: msc.CompanyID},
		{name: "华南-欧洲快线 (Guangzhou-Rotterdam)", depPort: "Guangzhou Nansha Port", destPort: "Rotterdam Port", desc: "广州直航鹿特丹快线", seq: "[1,6,11]", dist: 7200, cid: cosco.CompanyID},
		{name: "东北亚支线 (Shanghai-Hong Kong)", depPort: "Shanghai Yangshan Port", destPort: "Hong Kong Port", desc: "中国沿海支线", seq: "[2,4,5,3]", dist: 2800, cid: cosco.CompanyID},
		{name: "泛太平洋-南美航线 (Shanghai-Santos)", depPort: "Shanghai Yangshan Port", destPort: "Santos Port", desc: "跨太平洋至巴西", seq: "[2,16,15,19,13]", dist: 12000, cid: cosco.CompanyID},
		{name: "地中海-中国航线 (Piraeus-Guangzhou)", depPort: "Piraeus Port", destPort: "Guangzhou Nansha Port", desc: "地中海经迪拜至中国", seq: "[18,17,6,1]", dist: 7800, cid: msc.CompanyID},
		{name: "欧洲-北美西海岸航线 (Rotterdam-Long Beach)", depPort: "Rotterdam Port", destPort: "Long Beach Port", desc: "跨大西洋至美西", seq: "[11,12,19,20]", dist: 9600, cid: maersk.CompanyID},
		{name: "中国-非洲航线 (Ningbo-Durban)", depPort: "Ningbo Zhoushan Port", destPort: "Durban Port", desc: "中国至南部非洲", seq: "[5,3,6,8,9]", dist: 8500, cid: cma.CompanyID},
		{name: "亚欧航线 (Shenzhen-Rotterdam)", depPort: "Shenzhen Yantian Port", destPort: "Rotterdam Port", desc: "深圳至欧洲主力航线", seq: "[4,6,17,18,11]", dist: 8200, cid: cma.CompanyID},
		{name: "亚洲-南美西海岸 (Hong Kong-LA)", depPort: "Hong Kong Port", destPort: "Los Angeles Port", desc: "亚洲至美西", seq: "[3,6,15,19]", dist: 10500, cid: cma.CompanyID},
		{name: "中东-非洲航线 (Dubai-Cape Town)", depPort: "Dubai Port", destPort: "Cape Town Port", desc: "中东至南部非洲", seq: "[17,6,8,10]", dist: 6000, cid: maersk.CompanyID},
	}
	lines := make([]model.ShippingLine, 0)
	for _, ld := range linesData {
		s := ld.seq
		lines = append(lines, model.ShippingLine{
			LineName: ld.name, ShippingCompanyID: &ld.cid, PortSequence: &s,
			TotalDistanceNm: f64Ptr(ld.dist), DeparturePortName: strPtr(ld.depPort),
			DestinationPortName: strPtr(ld.destPort), Description: strPtr(ld.desc),
			LineStatus: ptrInt8(model.LineStatusActive), CreateTime: now, UpdateTime: now,
		})
	}
	for i := range lines { mustCreate(db, &lines[i]) }
	fmt.Printf("Created %d shipping lines\n", len(lines))

	// ===== Line-Vessel assignments =====
	for _, l := range lines {
		for _, vi := range getCompanyVessels(*l.ShippingCompanyID) {
			mustCreate(db, &model.LineVessel{LineID: l.LineID, VesselID: vessels[vi].VesselID, CreateTime: now})
		}
	}
	fmt.Println("Assigned vessels to lines")

	// ===== Parse port sequences for voyage generation =====
	type seqInfo struct{ portIDs []int64; portIndices []int }
	lineSeqs := make([]seqInfo, len(lines))
	for li, l := range lines {
		var ids []int64
		if err := parseJSON(*l.PortSequence, &ids); err != nil {
			log.Fatalf("parse port sequence for line %d: %v", li, err)
		}
		idxs := make([]int, len(ids))
		for j, pid := range ids {
			for pi, p := range ports {
				if p.PortID == pid { idxs[j] = pi; break }
			}
		}
		lineSeqs[li] = seqInfo{portIDs: ids, portIndices: idxs}
	}

	// ===== Voyages (programmatic, large scale) =====
	type voyagePlan struct {
		lineIdx    int
		vesselIdx  int
		date       time.Time
		unitPrice  float64
	}
	voyagePlans := make([]voyagePlan, 0)
	cargoTypeCodes := []string{"bulk", "container", "liquid", "reefer", "dangerous"}

	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for li, l := range lines {
		companyVessels := getCompanyVessels(*l.ShippingCompanyID)
		numVoyages := 3 + rng.Intn(3) // 3-5 voyages per line
		for v := 0; v < numVoyages; v++ {
			vi := companyVessels[rng.Intn(len(companyVessels))]
			daysOffset := rng.Intn(240) // spread across 8 months
			d := startDate.AddDate(0, 0, daysOffset)
			up := 60.0 + rng.Float64()*140.0
			voyagePlans = append(voyagePlans, voyagePlan{
				lineIdx: li, vesselIdx: vi, date: d, unitPrice: up,
			})
		}
	}
	fmt.Printf("Planning %d voyages\n", len(voyagePlans))

	type createdVoyage struct {
		lineIdx   int
		vesselIdx int
		date      time.Time
		startPort int
		endPort   int
	}
	createdVoyages := make([]createdVoyage, 0)

	for _, vp := range voyagePlans {
		l := lines[vp.lineIdx]
		ves := vessels[vp.vesselIdx]
		seq := lineSeqs[vp.lineIdx]
		numStops := len(seq.portIndices)

		voyageDate := vp.date
		// create berthing for each port in sequence
		for seqIdx := 0; seqIdx < numStops; seqIdx++ {
			portIdx := seq.portIndices[seqIdx]
			arriveD := 1 + seqIdx*3
			departD := arriveD + 1
			if departD > 28 { departD = 28 }
			arriveH := 6 + rng.Intn(4)
			departH := arriveH + 8 + rng.Intn(4)

			vb := model.VoyageBerthing{
				LineID: &l.LineID, VesselID: &ves.VesselID, VoyageDate: voyageDate,
				SequenceNo: int32(seqIdx + 1), PortID: &ports[portIdx].PortID,
				PlannedArrivalTime:   timePtr(time.Date(voyageDate.Year(), voyageDate.Month(), arriveD, arriveH, 0, 0, 0, time.UTC)),
				PlannedDepartureTime: timePtr(time.Date(voyageDate.Year(), voyageDate.Month(), departD, departH, 0, 0, 0, time.UTC)),
				IsAdjustable: 1, CreateTime: now, UpdateTime: now,
			}
			mustCreate(db, &vb)

			op := "LOAD"
			if seqIdx == numStops-1 { op = "UNLOAD" }
			cn := "待定"
			ct := "bulk"
			z := 0.0
			mustCreate(db, &model.VoyageCargoNote{
				LineID: &l.LineID, VesselID: &ves.VesselID, VoyageDate: voyageDate,
				SequenceNo: int32(seqIdx + 1), CargoName: &cn, CargoType: &ct, OperationType: &op,
				UnitPrice: &vp.unitPrice,
				Quantity: &z, WeightTon: &z, VolumeCubicMeter: &z, Subtotal: &z,
				CreateTime: now, UpdateTime: now,
			})
		}

		createdVoyages = append(createdVoyages, createdVoyage{
			lineIdx: vp.lineIdx, vesselIdx: vp.vesselIdx, date: voyageDate,
			startPort: seq.portIndices[0], endPort: seq.portIndices[numStops-1],
		})
	}
	fmt.Printf("Created %d voyages with berthing and cargo notes\n", len(createdVoyages))

	// ===== Specific cargo notes for a few voyages (so orders can link) =====
	for i := 0; i < len(createdVoyages) && i < 8; i++ {
		cv := createdVoyages[i]
		l := lines[cv.lineIdx]
		ves := vessels[cv.vesselIdx]
		seq := lineSeqs[cv.lineIdx]
		cn := cargonames[rng.Intn(len(cargonames))]
		ct := cargoTypeCodes[rng.Intn(len(cargoTypeCodes))]
		q := 50.0 + rng.Float64()*200.0
		w := q * (0.8 + rng.Float64()*0.4)
		v := w * (1.0 + rng.Float64())
		up := 60.0 + rng.Float64()*140.0
		sub := q * up

		// LOAD at first port
		mustCreate(db, &model.VoyageCargoNote{
			LineID: &l.LineID, VesselID: &ves.VesselID, VoyageDate: cv.date,
			SequenceNo: 1, CargoName: strPtr(cn), CargoType: strPtr(ct),
			Quantity: &q, WeightTon: &w, VolumeCubicMeter: &v, UnitPrice: &up, Subtotal: &sub,
			OperationType: strPtr("LOAD"), CargoHandledTon: &w, CumulativeBookedCapacityTon: &w,
			CreateTime: now, UpdateTime: now,
		})
		// UNLOAD at last port
		lastSeq := int32(len(seq.portIndices))
		w2 := w * 0.6
		v2 := w2 * (1.0 + rng.Float64())
		q2 := q * 0.6
		sub2 := q2 * up
		mustCreate(db, &model.VoyageCargoNote{
			LineID: &l.LineID, VesselID: &ves.VesselID, VoyageDate: cv.date,
			SequenceNo: lastSeq, CargoName: strPtr(cn), CargoType: strPtr(ct),
			Quantity: &q2, WeightTon: &w2, VolumeCubicMeter: &v2, UnitPrice: &up, Subtotal: &sub2,
			OperationType: strPtr("UNLOAD"), CargoHandledTon: &w2,
			CreateTime: now, UpdateTime: now,
		})
	}
	fmt.Println("Created specific cargo notes for orders")

	// ===== Cargo Types =====
	ctList := []model.CargoType{
		{TypeName: "散货", TypeCode: "bulk", Description: strPtr("散装货物，如矿石、煤炭、谷物等")},
		{TypeName: "集装箱", TypeCode: "container", Description: strPtr("标准集装箱货物")},
		{TypeName: "液体", TypeCode: "liquid", Description: strPtr("液体货物，如石油、化工原料等")},
		{TypeName: "冷藏货", TypeCode: "reefer", Description: strPtr("需要冷藏运输的货物")},
		{TypeName: "危险品", TypeCode: "dangerous", Description: strPtr("危险货物，需特殊标识和操作")},
		{TypeName: "超大件", TypeCode: "oversized", Description: strPtr("超长超重超高超宽货物")},
	}
	for i := range ctList { ctList[i].CreateTime = now; ctList[i].UpdateTime = now; mustCreate(db, &ctList[i]) }
	fmt.Println("Created 6 cargo types")

	// ===== Orders (large scale) =====
	type orderGen struct {
		no        string
		shipperIdx int
		cityIdx    int
		depP       int
		destP      int
		depDate    string
		arrDate    string
		cargoName  string
		cargoType  string
		qty        float64
		weight     float64
		volume     float64
		unitPrice  float64
		payStatus  int8
		orderStatus int8
	}
	orderList := make([]orderGen, 0)

	// 5 orders per voyage, spread across different voyages
	for vi, cv := range createdVoyages {
		if vi%3 != 0 { continue } // every 3rd voyage gets orders
		seq := lineSeqs[cv.lineIdx]
		if len(seq.portIndices) < 2 { continue }

		numOrders := 2 + rng.Intn(4) // 2-5 orders per voyage
		for o := 0; o < numOrders; o++ {
			si := rng.Intn(len(shippers))
			ci := rng.Intn(len(cities))
			startIdx := 0
			endIdx := len(seq.portIndices) - 1
			if endIdx > startIdx+1 {
				endIdx = startIdx + 1 + rng.Intn(endIdx-startIdx)
			}
			depP := seq.portIndices[startIdx]
			destP := seq.portIndices[endIdx]
			depDate := cv.date.Format("2006-01-02")
			arrDate := cv.date.AddDate(0, 0, 10+rng.Intn(20)).Format("2006-01-02")
			cn := cargonames[rng.Intn(len(cargonames))]
			ct := cargoTypeCodes[rng.Intn(len(cargoTypeCodes))]
			q := 20.0 + rng.Float64()*100.0
			w := q * (0.8 + rng.Float64()*0.4)
			v := w * (1.0 + rng.Float64())
			up := 60.0 + rng.Float64()*140.0
			status := int8(rng.Intn(4)) // 0-3
			pay := int8(0)
			if status >= 1 { pay = 1 }

			orderNo := fmt.Sprintf("ORD%s%08x", cv.date.Format("20060106"), len(orderList)+1)
			orderList = append(orderList, orderGen{
				no: orderNo, shipperIdx: si, cityIdx: ci,
				depP: depP, destP: destP,
				depDate: depDate, arrDate: arrDate,
				cargoName: cn, cargoType: ct, qty: q, weight: w, volume: v, unitPrice: up,
				payStatus: pay, orderStatus: status,
			})
		}
	}

	// Also generate some orders for the first few voyages specifically
	// to ensure the original demo data is there
	fmt.Printf("Planning %d orders\n", len(orderList))

	for _, od := range orderList {
		depDate, _ := time.Parse("2006-01-02", od.depDate)
		arrDate, _ := time.Parse("2006-01-02", od.arrDate)

		// Find a voyage_cargo_note to link this order to
		var loadNote model.VoyageCargoNote
		loadNoteID := (*int64)(nil)
		portID := ports[od.depP].PortID
		if err := db.Table("voyage_cargo_note").
			Select("voyage_cargo_note.note_id").
			Joins("JOIN voyage_berthing ON voyage_berthing.line_id = voyage_cargo_note.line_id AND voyage_berthing.vessel_id = voyage_cargo_note.vessel_id AND voyage_berthing.voyage_date = voyage_cargo_note.voyage_date AND voyage_berthing.sequence_no = voyage_cargo_note.sequence_no").
			Where("voyage_berthing.port_id = ?", portID).
			Limit(1).Scan(&loadNote).Error; err == nil && loadNote.NoteID > 0 {
			loadNoteID = &loadNote.NoteID
		}

		subtotal := od.qty * od.unitPrice
		cost := od.weight * od.unitPrice
		order := model.ShippingOrder{
			OrderNo: od.no, ShipperCompanyID: &shippers[od.shipperIdx].CompanyID,
			CityID: &cities[od.cityIdx].CityID,
			DeparturePortID: &ports[od.depP].PortID, DestinationPortID: &ports[od.destP].PortID,
			LoadNoteID: loadNoteID,
			ExpectedDepartureDate: &depDate, ExpectedArrivalDate: &arrDate,
			TotalCost: &cost, ShipperContact: strPtr(fmt.Sprintf("联系人-%s", shippers[od.shipperIdx].CompanyName)),
			ConsigneeContact: strPtr(fmt.Sprintf("收货方-%d", rng.Intn(1000))),
			PaymentStatus: &od.payStatus, OrderStatus: &od.orderStatus,
			TotalWeightTon: &od.weight, TotalVolumeCubicMeter: &od.volume,
			CreateTime: now, UpdateTime: now,
		}
		mustCreate(db, &order)

		mustCreate(db, &model.OrderCargo{
			OrderID: &order.OrderID, CargoName: strPtr(od.cargoName), CargoType: strPtr(od.cargoType),
			Quantity: &od.qty, WeightTon: &od.weight, VolumeCubicMeter: &od.volume,
			UnitPrice: &od.unitPrice, Subtotal: &subtotal,
			CreateTime: now, UpdateTime: now,
		})
	}
	fmt.Printf("Created %d sample orders with cargo items\n", len(orderList))
}

var cargonames = []string{"铁矿石", "煤炭", "小麦", "钢材", "机械设备", "电子产品", "纺织品", "化工原料", "家具", "汽车零部件", "化肥", "塑料颗粒", "纸浆", "橡胶", "水泥"}

func mustCreate(db *gorm.DB, value interface{}) {
	if err := db.Create(value).Error; err != nil {
		log.Fatalf("failed to create: %v", err)
	}
}

func strPtr(s string) *string { return &s }
func strPtrOrNil(s string) *string { if s == "" { return nil }; return &s }
func f64Ptr(f float64) *float64 { return &f }
func timePtr(t time.Time) *time.Time { return &t }
func ptrInt8(v int8) *int8 { return &v }

func parseJSON(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}
