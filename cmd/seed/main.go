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
	fmt.Println("Config loaded, connecting to database...")

	db := database.MustNewMySQL(cfg.Database, "info", 200*time.Millisecond)
	fmt.Println("Database connected, starting seed...")

	cleanDB(db)
	seed(db)

	fmt.Println("Seed completed successfully!")
}

func cleanDB(db *gorm.DB) {
	tables := []string{
		"segment_capacity_usage",
		"order_cargo",
		"shipping_order",
		"voyage_cargo_note",
		"voyage_berthing",
		"shipping_line",
		"vessel",
		"berth",
		"port",
		"city",
		"shipper_company",
		"shipping_company",
		"admin",
	}
	for _, t := range tables {
		db.Exec("DELETE FROM " + t)
		db.Exec("ALTER TABLE " + t + " AUTO_INCREMENT = 1")
	}
	fmt.Println("Cleaned all tables")
}

func seed(db *gorm.DB) {
	now := time.Now()

	// ===== Cities =====
	shanghai := model.City{
		CityName:    "Shanghai",
		Country:     strPtr("China"),
		CountryCode: strPtr("CN"),
		Timezone:    strPtr("Asia/Shanghai"),
		Latitude:    f64Ptr(31.2304),
		Longitude:   f64Ptr(121.4737),
		CreateTime:  now,
		UpdateTime:  now,
	}
	singapore := model.City{
		CityName:    "Singapore",
		Country:     strPtr("Singapore"),
		CountryCode: strPtr("SG"),
		Timezone:    strPtr("Asia/Singapore"),
		Latitude:    f64Ptr(1.3521),
		Longitude:   f64Ptr(103.8198),
		CreateTime:  now,
		UpdateTime:  now,
	}
	rotterdam := model.City{
		CityName:    "Rotterdam",
		Country:     strPtr("Netherlands"),
		CountryCode: strPtr("NL"),
		Timezone:    strPtr("Europe/Amsterdam"),
		Latitude:    f64Ptr(51.9225),
		Longitude:   f64Ptr(4.4792),
		CreateTime:  now,
		UpdateTime:  now,
	}
	mustCreate(db, &shanghai)
	mustCreate(db, &singapore)
	mustCreate(db, &rotterdam)
	fmt.Println("Cities created")

	// ===== Ports =====
	portSH := model.Port{
		PortName:      "Shanghai Port",
		PortCode:      strPtr("CNSHA"),
		CityID:        &shanghai.CityID,
		Latitude:      f64Ptr(31.2304),
		Longitude:     f64Ptr(121.4737),
		PortType:      strPtr("Sea Port"),
		MaxDraftMeter: f64Ptr(15.5),
		CreateTime:    now,
		UpdateTime:    now,
	}
	portSG := model.Port{
		PortName:      "Singapore Port",
		PortCode:      strPtr("SGSIN"),
		CityID:        &singapore.CityID,
		Latitude:      f64Ptr(1.2645),
		Longitude:     f64Ptr(103.8390),
		PortType:      strPtr("Sea Port"),
		MaxDraftMeter: f64Ptr(16.0),
		CreateTime:    now,
		UpdateTime:    now,
	}
	portRT := model.Port{
		PortName:      "Rotterdam Port",
		PortCode:      strPtr("NLRTM"),
		CityID:        &rotterdam.CityID,
		Latitude:      f64Ptr(51.9052),
		Longitude:     f64Ptr(4.4646),
		PortType:      strPtr("Sea Port"),
		MaxDraftMeter: f64Ptr(17.0),
		CreateTime:    now,
		UpdateTime:    now,
	}
	mustCreate(db, &portSH)
	mustCreate(db, &portSG)
	mustCreate(db, &portRT)
	fmt.Println("Ports created")

	// ===== Berths =====
	berthSH := model.Berth{
		BerthName:          "Shanghai Berth A",
		PortID:             &portSH.PortID,
		BerthType:          strPtr("Container"),
		DraftMeter:         f64Ptr(14.0),
		LengthMeter:        f64Ptr(300),
		WidthMeter:         f64Ptr(50),
		MaxBerthingTonnage: f64Ptr(80000),
		FunctionalZone:     strPtr("Zone A"),
		IsAvailable:        1,
		CreateTime:         now,
		UpdateTime:         now,
	}
	berthSG := model.Berth{
		BerthName:          "Singapore Berth B",
		PortID:             &portSG.PortID,
		BerthType:          strPtr("Container"),
		DraftMeter:         f64Ptr(15.0),
		LengthMeter:        f64Ptr(350),
		WidthMeter:         f64Ptr(55),
		MaxBerthingTonnage: f64Ptr(100000),
		FunctionalZone:     strPtr("Zone B"),
		IsAvailable:        1,
		CreateTime:         now,
		UpdateTime:         now,
	}
	berthRT := model.Berth{
		BerthName:          "Rotterdam Berth C",
		PortID:             &portRT.PortID,
		BerthType:          strPtr("Container"),
		DraftMeter:         f64Ptr(16.0),
		LengthMeter:        f64Ptr(400),
		WidthMeter:         f64Ptr(60),
		MaxBerthingTonnage: f64Ptr(120000),
		FunctionalZone:     strPtr("Zone C"),
		IsAvailable:        1,
		CreateTime:         now,
		UpdateTime:         now,
	}
	mustCreate(db, &berthSH)
	mustCreate(db, &berthSG)
	mustCreate(db, &berthRT)
	fmt.Println("Berths created")

	// ===== Shipper Company =====
	shipperPass, err := crypto.HashPassword("123456")
	if err != nil {
		log.Fatalf("failed to hash shipper password: %v", err)
	}
	shipper := model.ShipperCompany{
		CompanyName:   "Global Trade Co.",
		LoginUsername: "test001",
		LoginPassword: shipperPass,
		AccountStatus: 1,
		CreateTime:    now,
		UpdateTime:    now,
	}
	mustCreate(db, &shipper)
	fmt.Println("Shipper company created (test001 / 123456)")

	// ===== Shipping Company =====
	shippingPass, err := crypto.HashPassword("123456")
	if err != nil {
		log.Fatalf("failed to hash shipping password: %v", err)
	}
	shipping := model.ShippingCompany{
		CompanyName:      "Oceanic Shipping Co.",
		UnifiedSocialCreditCode: strPtr("SHIP20240001"),
		ContactPerson:    strPtr("John Smith"),
		ContactPhone:     strPtr("+65-12345678"),
		Address:          strPtr("12 Harbor Road, Singapore"),
		LoginUsername:    "shipping001",
		LoginPassword:    shippingPass,
		AccountStatus:    1,
		CreateTime:       now,
		UpdateTime:       now,
	}
	mustCreate(db, &shipping)
	fmt.Println("Shipping company created (shipping001 / 123456)")

	// ===== Admin =====
	adminPass, err := crypto.HashPassword("admin123")
	if err != nil {
		log.Fatalf("failed to hash admin password: %v", err)
	}
	admin := model.Admin{
		Username:   "admin",
		Password:   adminPass,
		RealName:   strPtr("System Admin"),
		Role:       1,
		CreateTime: now,
		UpdateTime: now,
	}
	mustCreate(db, &admin)
	fmt.Println("Admin created (admin / admin123)")

	// ===== Vessel =====
	containerTEU := int32(5000)
	vessel := model.Vessel{
		VesselName:        "M/V Ocean Queen",
		CallSign:          strPtr("OCQN"),
		IMONumber:         "IMO9876543",
		VesselType:        strPtr("Container Ship"),
		MaxDeadweightTon:  f64Ptr(50000),
		GrossTonnage:      f64Ptr(35000),
		NetTonnage:        f64Ptr(25000),
		DraftMeter:        f64Ptr(12.5),
		SpeedKnot:         f64Ptr(22),
		ContainerTEU:      &containerTEU,
		IsAvailable:       1,
		ShippingCompanyID: &shipping.CompanyID,
		CreateTime:        now,
		UpdateTime:        now,
	}
	mustCreate(db, &vessel)
	fmt.Println("Vessel created")

	// ===== Shipping Line =====
	portSeq := `[1,2,3]`
	line := model.ShippingLine{
		LineName:            "Asia-Europe Express",
		ShippingCompanyID:   &shipping.CompanyID,
		PortSequence:        &portSeq,
		TotalDistanceNm:     f64Ptr(8500),
		DeparturePortName:   strPtr("Shanghai Port"),
		DestinationPortName: strPtr("Rotterdam Port"),
		Description:         strPtr("Asia to Europe direct route via Singapore"),
		CreateTime:          now,
		UpdateTime:          now,
	}
	mustCreate(db, &line)
	fmt.Println("Shipping line created")

	// ===== Voyage Berthing =====
	voyageDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)

	vb1 := model.VoyageBerthing{
		LineID:               &line.LineID,
		VesselID:             &vessel.VesselID,
		VoyageDate:           voyageDate,
		SequenceNo:           1,
		PortID:               &portSH.PortID,
		BerthID:              &berthSH.BerthID,
		PlannedArrivalTime:   timePtr(time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)),
		PlannedDepartureTime: timePtr(time.Date(2026, 7, 15, 18, 0, 0, 0, time.UTC)),
		IsAdjustable:         1,
		CreateTime:           now,
		UpdateTime:           now,
	}
	vb2 := model.VoyageBerthing{
		LineID:               &line.LineID,
		VesselID:             &vessel.VesselID,
		VoyageDate:           voyageDate,
		SequenceNo:           2,
		PortID:               &portSG.PortID,
		BerthID:              &berthSG.BerthID,
		PlannedArrivalTime:   timePtr(time.Date(2026, 7, 17, 6, 0, 0, 0, time.UTC)),
		PlannedDepartureTime: timePtr(time.Date(2026, 7, 17, 16, 0, 0, 0, time.UTC)),
		IsAdjustable:         1,
		CreateTime:           now,
		UpdateTime:           now,
	}
	vb3 := model.VoyageBerthing{
		LineID:               &line.LineID,
		VesselID:             &vessel.VesselID,
		VoyageDate:           voyageDate,
		SequenceNo:           3,
		PortID:               &portRT.PortID,
		BerthID:              &berthRT.BerthID,
		PlannedArrivalTime:   timePtr(time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)),
		PlannedDepartureTime: timePtr(time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC)),
		IsAdjustable:         1,
		CreateTime:           now,
		UpdateTime:           now,
	}
	mustCreate(db, &vb1)
	mustCreate(db, &vb2)
	mustCreate(db, &vb3)
	fmt.Println("Voyage berthing records created")

	// ===== Voyage Cargo Notes =====
	// LOAD at Shanghai (seq 1)
	qty1 := 1000.0
	wgt1 := 500.0
	vol1 := 2500.0
	up1 := 150.0
	sub1 := qty1 * up1
	note1 := model.VoyageCargoNote{
		LineID:           &line.LineID,
		VesselID:         &vessel.VesselID,
		VoyageDate:       voyageDate,
		SequenceNo:       1,
		CargoName:        strPtr("Electronics"),
		CargoType:        strPtr("container"),
		Quantity:         &qty1,
		WeightTon:        &wgt1,
		VolumeCubicMeter: &vol1,
		UnitPrice:        &up1,
		Subtotal:         &sub1,
		OperationType:    strPtr("LOAD"),
		CargoHandledTon:  &wgt1,
		CreateTime:       now,
		UpdateTime:       now,
	}

	// UNLOAD at Rotterdam (seq 3)
	qty2 := 800.0
	wgt2 := 400.0
	vol2 := 2000.0
	up2 := 200.0
	sub2 := qty2 * up2
	note2 := model.VoyageCargoNote{
		LineID:           &line.LineID,
		VesselID:         &vessel.VesselID,
		VoyageDate:       voyageDate,
		SequenceNo:       3,
		CargoName:        strPtr("Machinery"),
		CargoType:        strPtr("container"),
		Quantity:         &qty2,
		WeightTon:        &wgt2,
		VolumeCubicMeter: &vol2,
		UnitPrice:        &up2,
		Subtotal:         &sub2,
		OperationType:    strPtr("UNLOAD"),
		CargoHandledTon:  &wgt2,
		CreateTime:       now,
		UpdateTime:       now,
	}

	mustCreate(db, &note1)
	mustCreate(db, &note2)
	fmt.Println("Voyage cargo notes created")
}

func mustCreate(db *gorm.DB, value interface{}) {
	if err := db.Create(value).Error; err != nil {
		log.Fatalf("failed to create: %v", err)
	}
}

func strPtr(s string) *string {
	return &s
}

func f64Ptr(f float64) *float64 {
	return &f
}

func timePtr(t time.Time) *time.Time {
	return &t
}
