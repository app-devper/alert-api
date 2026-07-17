package domain

import (
	"context"
	"time"

	"alert/app/core/constant"
	"alert/app/data/entities"
	"alert/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var defaultTemplateTexts = map[string]entities.ChannelText{
	constant.EventFire: {
		TextTh: "แจ้งเหตุฉุกเฉิน: พบเหตุเพลิงไหม้ กรุณาหยุดกิจกรรมและออกจากร้านทันทีทางประตูฉุกเฉินที่ใกล้ที่สุด ห้ามใช้ลิฟต์ และปฏิบัติตามคำแนะนำของพนักงาน",
		TextEn: "FIRE ALERT: Fire reported. Please stop all activity and exit immediately via the nearest emergency exit. Do not use elevators. Follow staff instructions.",
	},
	constant.EventEvacuate: {
		TextTh: "แจ้งเหตุฉุกเฉิน: กรุณาออกจากร้านทันทีอย่างเป็นระเบียบ และปฏิบัติตามคำแนะนำของพนักงาน",
		TextEn: "EMERGENCY: Please evacuate the premises immediately in an orderly manner and follow staff instructions.",
	},
	constant.EventAvoidArea: {
		TextTh: "ประกาศ: เกิดเหตุในบางพื้นที่ของร้าน กรุณาหลีกเลี่ยงบริเวณที่เกิดเหตุ และปฏิบัติตามคำแนะนำของพนักงาน",
		TextEn: "NOTICE: An incident has occurred in part of the venue. Please avoid the affected area and follow staff instructions.",
	},
	constant.EventSuspiciousObject: {
		TextTh: "แจ้งเตือน: พบวัตถุต้องสงสัยภายในร้าน กรุณาอย่าเข้าใกล้หรือสัมผัส และปฏิบัติตามคำแนะนำของพนักงาน",
		TextEn: "ALERT: A suspicious object has been found. Do not approach or touch it. Follow staff instructions.",
	},
	constant.EventBrawl: {
		TextTh: "แจ้งเตือน: เกิดเหตุทะเลาะวิวาทภายในร้าน กรุณาหลีกเลี่ยงบริเวณที่เกิดเหตุเพื่อความปลอดภัยของท่าน",
		TextEn: "ALERT: An altercation is occurring. Please stay clear of the affected area for your safety.",
	},
	constant.EventExternal: {
		TextTh: "ประกาศ: เกิดเหตุภายนอกร้าน เพื่อความปลอดภัย กรุณาอยู่ภายในร้านจนกว่าจะมีประกาศเพิ่มเติม",
		TextEn: "NOTICE: An incident is occurring outside the venue. For your safety, please remain inside until further notice.",
	},
	constant.EventAllClear: {
		TextTh: "ประกาศ: เหตุการณ์กลับสู่ภาวะปกติแล้ว ขออภัยในความไม่สะดวก และขอบคุณที่ให้ความร่วมมือ",
		TextEn: "ALL CLEAR: The situation has returned to normal. We apologize for the inconvenience and thank you for your cooperation.",
	},
	constant.EventTest: {
		TextTh: "[TEST] ข้อความทดสอบระบบแจ้งเตือนฉุกเฉิน ไม่ใช่เหตุการณ์จริง ไม่ต้องดำเนินการใด ๆ",
		TextEn: "[TEST] This is a test of the emergency alert system. No action is required.",
	},
}

func TemplateSeeder() db.Seeder {
	return func(ctx context.Context, clientId string, database *mongo.Database) error {
		col := database.Collection(db.CollectionMessageTemplates)
		for _, code := range constant.EventTypes {
			count, err := col.CountDocuments(ctx, bson.M{"clientId": clientId, "code": code, "active": true})
			if err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			texts := defaultTemplateTexts[code]
			template := entities.MessageTemplate{
				Id:        primitive.NewObjectID(),
				ClientId:  clientId,
				Code:      code,
				TextTh:    texts.TextTh,
				TextEn:    texts.TextEn,
				Active:    true,
				UpdatedBy: "SYSTEM",
				UpdatedAt: time.Now(),
			}
			if _, err := col.InsertOne(ctx, template); err != nil {
				return err
			}
		}
		return nil
	}
}
