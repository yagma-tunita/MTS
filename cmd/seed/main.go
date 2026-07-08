package main

import (
	"fmt"
	"log"
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
	// 确保新字段存在
	db.Exec("ALTER TABLE shipping_line ADD COLUMN line_status TINYINT DEFAULT 1 AFTER description")
	tables := []string{
		"segment_capacity_usage", "order_cargo", "shipping_order",
		"voyage_cargo_note", "voyage_berthing", "shipping_line",
		"vessel", "berth", "port", "city",
		"shipper_company", "shipping_company", "admin",
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
	sp1, _ := crypto.HashPassword("123456")
	mustCreate(db, &model.ShipperCompany{
		CompanyName: "中国远洋贸易有限公司", UnifiedSocialCreditCode: strPtr("91440101MA5XXXXXX1"),
		LegalRepresentative: strPtr("张伟"), ContactPhone: strPtr("13800138001"),
		Address: strPtr("广州天河区黄埔大道西100号"), LoginUsername: "shipper01", LoginPassword: sp1,
		AccountStatus: 1, CreateTime: now, UpdateTime: now,
	})
	sp2, _ := crypto.HashPassword("123456")
	mustCreate(db, &model.ShipperCompany{
		CompanyName: "上海国际矿业资源公司", UnifiedSocialCreditCode: strPtr("91310000MA5XXXXXX2"),
		LegalRepresentative: strPtr("李强"), ContactPhone: strPtr("13800138002"),
		Address: strPtr("上海浦东新区陆家嘴金融区100号"), LoginUsername: "shipper02", LoginPassword: sp2,
		AccountStatus: 1, CreateTime: now, UpdateTime: now,
	})
	sp3, _ := crypto.HashPassword("123456")
	mustCreate(db, &model.ShipperCompany{
		CompanyName: "深圳华远物流有限公司", LegalRepresentative: strPtr("王明"),
		ContactPhone: strPtr("13800138006"), Address: strPtr("深圳南山区蛇口工业区"),
		LoginUsername: "shipper03", LoginPassword: sp3, AccountStatus: 1, CreateTime: now, UpdateTime: now,
	})
	fmt.Println("Created 3 shippers (shipper01/02/03, password: 123456)")

	// ===== Shipping Companies =====
	co1, _ := crypto.HashPassword("123456")
	cosco := &model.ShippingCompany{
		CompanyName: "中国远洋海运集团", ContactPerson: strPtr("王建国"), ContactPhone: strPtr("13800138003"),
		Address: strPtr("北京朝阳区建国路100号"), LoginUsername: "cosco", LoginPassword: co1,
		AccountStatus: 1, CreateTime: now, UpdateTime: now,
	}
	mustCreate(db, cosco)
	co2, _ := crypto.HashPassword("123456")
	maersk := &model.ShippingCompany{
		CompanyName: "马士基航运(中国)有限公司", ContactPerson: strPtr("Peter Jensen"),
		ContactPhone: strPtr("13800138004"), Address: strPtr("上海黄浦区南京西路100号"),
		LoginUsername: "maersk", LoginPassword: co2, AccountStatus: 1, CreateTime: now, UpdateTime: now,
	}
	mustCreate(db, maersk)
	co3, _ := crypto.HashPassword("123456")
	msc := &model.ShippingCompany{
		CompanyName: "地中海航运有限公司", ContactPerson: strPtr("陈志强"),
		ContactPhone: strPtr("13800138005"), Address: strPtr("深圳南山区蛇口工业区"),
		LoginUsername: "msc", LoginPassword: co3, AccountStatus: 1, CreateTime: now, UpdateTime: now,
	}
	mustCreate(db, msc)
	// 新增第四家海运公司
	co4, _ := crypto.HashPassword("123456")
	cma := &model.ShippingCompany{
		CompanyName: "达飞轮船(中国)有限公司", ContactPerson: strPtr("Jean Dupont"),
		ContactPhone: strPtr("13800138007"), Address: strPtr("上海静安区南京西路200号"),
		LoginUsername: "cma", LoginPassword: co4, AccountStatus: 1, CreateTime: now, UpdateTime: now,
	}
	mustCreate(db, cma)
	fmt.Println("Created 4 shipping companies (cosco/maersk/msc/cma, password: 123456)")

	// ===== Admin =====
	ap, _ := crypto.HashPassword("admin123")
	mustCreate(db, &model.Admin{Username: "admin", Password: ap, RealName: strPtr("系统管理员"), Role: 1, CreateTime: now, UpdateTime: now})
	fmt.Println("Created admin (admin/admin123)")

	// ===== 16 Vessels (4 per company) =====
	type vData struct{ name, call, imo, vtype string; dwt, speed float64; teu int32 }
	vesselsData := map[int64][]vData{
		cosco.CompanyID: {
			{"远洋号 (Yuan Yang Hao)", "BOCQ", "IMO1000001", "Bulk Carrier", 300, 14.5, 3000},
			{"东方巨龙 (Oriental Dragon)", "BODR", "IMO1000002", "Container Ship", 120000, 22.0, 8000},
			{"丝路号 (Silk Road)", "BSLR", "IMO1000006", "Container Ship", 90000, 21.0, 6000},
			{"郑和号 (Zheng He)", "BZHE", "IMO1000007", "Container Ship", 150000, 24.0, 10000},
		},
		maersk.CompanyID: {
			{"Maersk Guangzhou", "MGRZ", "IMO1000003", "Container Ship", 75000, 20.5, 5000},
			{"Maersk Shanghai", "MSH", "IMO1000009", "Container Ship", 180000, 23.5, 14000},
			{"Maersk Mumbai", "MMB", "IMO1000010", "Container Ship", 85000, 21.0, 6000},
			{"Maersk Copenhagen", "MCPH", "IMO1000013", "Container Ship", 95000, 22.0, 7000},
		},
		msc.CompanyID: {
			{"MSC Shanghai", "MSSH", "IMO1000005", "Container Ship", 180000, 23.0, 12000},
			{"MSC Tokyo", "MSTK", "IMO1000011", "Container Ship", 220000, 25.0, 20000},
			{"MSC Hamburg", "MSHB", "IMO1000012", "Oil Tanker", 200000, 16.5, 0},
			{"MSC Geneva", "MSG", "IMO1000014", "Container Ship", 130000, 22.5, 10000},
		},
		cma.CompanyID: {
			{"CMA CGM Paris", "CMAP", "IMO1000015", "Container Ship", 140000, 22.5, 11000},
			{"CMA CGM Marseille", "CMAM", "IMO1000016", "Container Ship", 110000, 21.5, 8000},
			{"CMA CGM Lyon", "CMAL", "IMO1000017", "Bulk Carrier", 60000, 15.5, 2000},
			{"CMA CGM Nice", "CMAN", "IMO1000018", "Container Ship", 170000, 23.0, 13000},
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
	fmt.Printf("Created %d vessels (4 per company)\n", len(vessels))

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
			LineStatus: model.LineStatusActive, CreateTime: now, UpdateTime: now,
		})
	}
	for i := range lines { mustCreate(db, &lines[i]) }
	fmt.Printf("Created %d shipping lines\n", len(lines))

	// ===== Voyages =====
	type voyageData struct{ lineIdx, vesselIdx int; date time.Time; stops []struct{ portIdx, arriveD, arriveT, departD, departT int } }
	voyages := []voyageData{
		{0, 0, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			[]struct{ portIdx, arriveD, arriveT, departD, departT int }{
				{0, 1, 6, 1, 18}, {5, 5, 8, 6, 18}, {6, 9, 10, 10, 20}, {8, 18, 7, 21, 17}, {12, 10, 8, 15, 12},
			}},
		{1, 1, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			[]struct{ portIdx, arriveD, arriveT, departD, departT int }{
				{1, 1, 6, 1, 20}, {5, 4, 9, 5, 21}, {16, 12, 7, 13, 19}, {17, 18, 7, 19, 15}, {10, 25, 10, 26, 18},
			}},
		{2, 4, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
			[]struct{ portIdx, arriveD, arriveT, departD, departT int }{
				{15, 15, 8, 16, 6}, {14, 17, 12, 18, 10}, {1, 20, 8, 21, 18}, {5, 24, 10, 25, 20}, {10, 5, 9, 6, 17},
			}},
		{4, 5, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			[]struct{ portIdx, arriveD, arriveT, departD, departT int }{
				{14, 1, 8, 2, 6}, {15, 3, 10, 4, 8}, {1, 6, 8, 7, 18}, {5, 10, 10, 11, 20}, {17, 20, 9, 21, 17},
			}},
		{5, 8, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			[]struct{ portIdx, arriveD, arriveT, departD, departT int }{
				{0, 1, 6, 1, 18}, {2, 2, 8, 2, 16}, {5, 5, 10, 6, 20}, {16, 12, 7, 13, 19}, {6, 18, 10, 19, 18},
			}},
		{6, 3, time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
			[]struct{ portIdx, arriveD, arriveT, departD, departT int }{
				{0, 15, 6, 15, 20}, {5, 18, 9, 19, 18}, {10, 28, 10, 29, 16},
			}},
		{7, 2, time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
			[]struct{ portIdx, arriveD, arriveT, departD, departT int }{
				{1, 10, 6, 10, 20}, {3, 11, 8, 11, 18}, {4, 12, 8, 12, 18}, {2, 13, 10, 13, 18},
			}},
		{9, 9, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			[]struct{ portIdx, arriveD, arriveT, departD, departT int }{
				{17, 1, 8, 1, 18}, {16, 5, 7, 6, 19}, {5, 10, 10, 11, 20}, {0, 15, 8, 15, 18},
			}},
		{10, 6, time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
			[]struct{ portIdx, arriveD, arriveT, departD, departT int }{
				{10, 10, 6, 10, 20}, {11, 12, 8, 12, 18}, {18, 20, 10, 21, 16},
			}},
		{11, 12, time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
			[]struct{ portIdx, arriveD, arriveT, departD, departT int }{
				{4, 20, 6, 20, 18}, {2, 21, 8, 21, 18}, {5, 24, 10, 25, 18}, {7, 29, 7, 30, 17}, {8, 5, 8, 6, 16},
			}},
	}
	for _, v := range voyages {
		vessel := vessels[v.vesselIdx]
		for seqIdx, s := range v.stops {
			makeTime := func(d, h int) *time.Time { return timePtr(time.Date(v.date.Year(), v.date.Month(), d, h, 0, 0, 0, time.UTC)) }
			vb := model.VoyageBerthing{
				LineID: &lines[v.lineIdx].LineID, VesselID: &vessel.VesselID, VoyageDate: v.date,
				SequenceNo: int32(seqIdx + 1), PortID: &ports[s.portIdx].PortID,
				PlannedArrivalTime: makeTime(s.arriveD, s.arriveT), PlannedDepartureTime: makeTime(s.departD, s.departT),
				IsAdjustable: 1, CreateTime: now, UpdateTime: now,
			}
			mustCreate(db, &vb)
			// auto-create cargo note for each berthing
			cn := "待定"; ct := "bulk"; op := "LOAD"; z := 0.0
			mustCreate(db, &model.VoyageCargoNote{
				LineID: &lines[v.lineIdx].LineID, VesselID: &vessel.VesselID, VoyageDate: v.date,
				SequenceNo: int32(seqIdx + 1), CargoName: &cn, CargoType: &ct, OperationType: &op,
				Quantity: &z, WeightTon: &z, VolumeCubicMeter: &z, UnitPrice: &z, Subtotal: &z,
				CreateTime: now, UpdateTime: now,
			})
		}
	}
	fmt.Println("Created 10 voyages with berthing and cargo notes")

	// ===== Additional cargo notes for specific voyages =====
	// Voyage 0 (line1, vessel0): iron ore load/unload
	q := 150.0; w := 150.0; v := 60.0; up := 85.0; sub := 150.0 * 85.0
	mustCreate(db, &model.VoyageCargoNote{
		LineID: &lines[0].LineID, VesselID: &vessels[0].VesselID, VoyageDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		SequenceNo: 1, CargoName: strPtr("铁矿石 (Iron Ore)"), CargoType: strPtr("bulk"),
		Quantity: &q, WeightTon: &w, VolumeCubicMeter: &v, UnitPrice: &up, Subtotal: &sub,
		OperationType: strPtr("LOAD"), CargoHandledTon: &w, CumulativeBookedCapacityTon: &w,
		CreateTime: now, UpdateTime: now,
	})
	q2 := 50.0; w2 := 50.0; v2 := 20.0; up2 := 85.0; sub2 := 50.0 * 85.0
	mustCreate(db, &model.VoyageCargoNote{
		LineID: &lines[0].LineID, VesselID: &vessels[0].VesselID, VoyageDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		SequenceNo: 4, CargoName: strPtr("铁矿石 (Iron Ore) - 德班卸"), CargoType: strPtr("bulk"),
		Quantity: &q2, WeightTon: &w2, VolumeCubicMeter: &v2, UnitPrice: &up2, Subtotal: &sub2,
		OperationType: strPtr("UNLOAD"), CargoHandledTon: &w2,
		CreateTime: now, UpdateTime: now,
	})
	q3 := 100.0; w3 := 100.0; v3 := 40.0; up3 := 85.0; sub3 := 100.0 * 85.0
	mustCreate(db, &model.VoyageCargoNote{
		LineID: &lines[0].LineID, VesselID: &vessels[0].VesselID, VoyageDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		SequenceNo: 5, CargoName: strPtr("铁矿石 (Iron Ore)"), CargoType: strPtr("bulk"),
		Quantity: &q3, WeightTon: &w3, VolumeCubicMeter: &v3, UnitPrice: &up3, Subtotal: &sub3,
		OperationType: strPtr("UNLOAD"), CargoHandledTon: &w3,
		CreateTime: now, UpdateTime: now,
	})
	// Voyage 1 (line2, vessel1): electronics
	q4 := 2000.0; w4 := 500.0; v4 := 2000.0; up4 := 120.0; sub4 := 2000.0 * 120.0
	mustCreate(db, &model.VoyageCargoNote{
		LineID: &lines[1].LineID, VesselID: &vessels[1].VesselID, VoyageDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		SequenceNo: 1, CargoName: strPtr("电子产品 (Electronics)"), CargoType: strPtr("container"),
		Quantity: &q4, WeightTon: &w4, VolumeCubicMeter: &v4, UnitPrice: &up4, Subtotal: &sub4,
		OperationType: strPtr("LOAD"), CargoHandledTon: &w4, CumulativeBookedCapacityTon: &w4,
		CreateTime: now, UpdateTime: now,
	})
	q5 := 1500.0; w5 := 400.0; v5 := 1500.0; up5 := 150.0; sub5 := 1500.0 * 150.0
	mustCreate(db, &model.VoyageCargoNote{
		LineID: &lines[1].LineID, VesselID: &vessels[1].VesselID, VoyageDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		SequenceNo: 5, CargoName: strPtr("电子产品 (Electronics)"), CargoType: strPtr("container"),
		Quantity: &q5, WeightTon: &w5, VolumeCubicMeter: &v5, UnitPrice: &up5, Subtotal: &sub5,
		OperationType: strPtr("UNLOAD"), CargoHandledTon: &w5,
		CreateTime: now, UpdateTime: now,
	})
	fmt.Println("Created specific cargo notes")

	// ===== 10 Sample Orders =====
	type odata struct {
		no string; sidx, cidx int; loadNote, unloadNote *model.VoyageCargoNote
		depP, destP int; depDate, arrDate string; cost, tw, tv float64
		contact, consignee string; payStatus int8; orderStatus int8
		cargoName, cargoType string; cq, cw, cv, cup, csub float64
	}
	ordersData := []odata{
		// 各订单分配不同状态：0=待确认, 1=已确认, 2=运输中, 3=已完成
		// 广州→德班：已支付、已确认（cosco 可见，可测试"发货"）
		{"ORD20260501a1b2c3d4", 0, 0, nil, nil, 0, 8, "2026-05-01", "2026-05-21", 50 * 85.0, 50, 20, "张伟-13800138001", "Durban Steel Works", 1, 1, "铁矿石 (Iron Ore)", "bulk", 50, 50, 20, 85, 4250},
		// 广州→桑托斯：未支付、待确认（cosco 可见，可测试"确认"）
		{"ORD20260501e5f6g7h8", 1, 0, nil, nil, 0, 12, "2026-05-01", "2026-06-15", 100 * 85.0, 100, 40, "李强-13800138002", "Santos Steel Mill", 0, 0, "铁矿石 (Iron Ore)", "bulk", 100, 100, 40, 85, 8500},
		// 上海→鹿特丹：已支付、运输中（cosco 可见，可测试"港口更新"）
		{"ORD20260701i9j0k1l2", 0, 1, nil, nil, 1, 10, "2026-07-01", "2026-07-26", 200 * 120.0, 200, 800, "张伟-13800138001", "Rotterdam Electronics BV", 1, 2, "智能手机 (Smartphones)", "container", 500, 50, 200, 120, 60000},
		// 东京→比雷埃夫斯：未支付、运输中（maersk 可见，可测试"港口更新"）
		{"ORD20260801m3n4o5p6", 1, 14, nil, nil, 14, 17, "2026-08-01", "2026-08-21", 200 * 85.0, 200, 400, "张伟-13800138001", "Piraeus Machinery Co.", 0, 2, "机械设备 (Machinery)", "bulk", 500, 200, 400, 150, 75000},
		// 广州→孟买：未支付、待确认（cosco 可见）
		{"ORD20260901q7r8s9t0", 1, 0, nil, nil, 0, 6, "2026-09-01", "2026-09-19", 300 * 85.0 * 1.2, 300, 800, "李强-13800138002", "Mumbai Electronics Ltd.", 0, 0, "消费电子 (Consumer Electronics)", "container", 1000, 300, 800, 200, 200000},
		// 广州→鹿特丹：未支付、待确认（cosco 可见）
		{"ORD20260815f1g2h3i4", 0, 0, nil, nil, 0, 10, "2026-08-15", "2026-08-29", 80 * 85.0, 80, 160, "王明-13800138006", "Rotterdam Steel Co.", 0, 0, "钢材 (Steel)", "bulk", 80, 80, 160, 85, 6800},
		// 上海→香港：已支付、运输中（cosco 可见，可测试"港口更新"）
		{"ORD20260710j5k6l7m8", 0, 1, nil, nil, 1, 2, "2026-07-10", "2026-07-13", 30 * 85.0, 30, 60, "王明-13800138006", "Hong Kong Trading Co.", 1, 2, "纺织品 (Textiles)", "container", 30, 30, 60, 120, 3600},
		// 比雷埃夫斯→广州：未支付、已确认（msc 可见，可测试"发货"）
		{"ORD20261001n9o0p1q2", 1, 17, nil, nil, 17, 0, "2026-10-01", "2026-10-15", 150 * 85.0, 150, 300, "李强-13800138002", "Guangzhou Import Co.", 0, 1, "化工原料 (Chemicals)", "bulk", 150, 150, 300, 85, 12750},
		// 鹿特丹→洛杉矶：已支付、已完成（maersk 可见）
		{"ORD20260910r3s4t5u6", 1, 10, nil, nil, 10, 18, "2026-09-10", "2026-09-21", 100 * 85.0 * 1.2, 100, 250, "张伟-13800138001", "LA Distribution Center", 1, 3, "家具 (Furniture)", "container", 100, 100, 250, 120, 12000},
		// 宁波→德班：未支付、待确认（cma 可见）
		{"ORD20260501v7w8x9y0", 2, 4, nil, nil, 4, 8, "2026-08-20", "2026-09-06", 60 * 85.0, 60, 120, "王明-13800138006", "Durban Logistics Co.", 0, 0, "机械设备 (Machinery)", "bulk", 60, 60, 120, 85, 5100},
	}
	for i, od := range ordersData {
		depDate, _ := time.Parse("2006-01-02", od.depDate)
		arrDate, _ := time.Parse("2006-01-02", od.arrDate)
		// Link order to a voyage cargo note at the same departure port,
		// so shipping companies can see it through load_note_id → voyage_cargo_note → shipping_line
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
		order := model.ShippingOrder{
			OrderNo: od.no, ShipperCompanyID: &shipperList(db, od.sidx).CompanyID, CityID: &cities[od.cidx].CityID,
			DeparturePortID: &ports[od.depP].PortID, DestinationPortID: &ports[od.destP].PortID,
			LoadNoteID: loadNoteID,
			ExpectedDepartureDate: &depDate, ExpectedArrivalDate: &arrDate,
			TotalCost: &od.cost, ShipperContact: strPtr(od.contact), ConsigneeContact: strPtr(od.consignee),
			PaymentStatus: &od.payStatus, OrderStatus: &od.orderStatus,
			TotalWeightTon: &od.tw, TotalVolumeCubicMeter: &od.tv,
			CreateTime: now, UpdateTime: now,
		}
		mustCreate(db, &order)
		mustCreate(db, &model.OrderCargo{
			OrderID: &order.OrderID, CargoName: strPtr(od.cargoName), CargoType: strPtr(od.cargoType),
			Quantity: &od.cq, WeightTon: &od.cw, VolumeCubicMeter: &od.cv, UnitPrice: &od.cup, Subtotal: &od.csub,
			CreateTime: now, UpdateTime: now,
		})
		if i == 2 {
			mustCreate(db, &model.OrderCargo{
				OrderID: &order.OrderID, CargoName: strPtr("笔记本电脑 (Laptops)"), CargoType: strPtr("container"),
				Quantity: f64Ptr(300), WeightTon: f64Ptr(150), VolumeCubicMeter: f64Ptr(600), UnitPrice: f64Ptr(120), Subtotal: f64Ptr(36000),
				CreateTime: now, UpdateTime: now,
			})
		}
	}
	fmt.Println("Created 10 sample orders with cargo items")
}

func shipperList(db *gorm.DB, idx int) *model.ShipperCompany {
	var companies []model.ShipperCompany
	db.Where("delete_time IS NULL").Order("company_id ASC").Find(&companies)
	if idx < len(companies) { return &companies[idx] }
	return &companies[0]
}

func mustCreate(db *gorm.DB, value interface{}) {
	if err := db.Create(value).Error; err != nil {
		log.Fatalf("failed to create: %v", err)
	}
}

func strPtr(s string) *string { return &s }
func f64Ptr(f float64) *float64 { return &f }
func timePtr(t time.Time) *time.Time { return &t }
