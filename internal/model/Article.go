package model

// 文章 model
type Article struct {
	ID      int    `gorm:"primary_key;auto_increment" json:"id"`
	Title   string `gorm:"type:varchar(100);not null" json:"title"`
	Cid     int    `gorm:"type:int;not null" json:"cid"`
	Desc    string `gorm:"type:varchar(200)" json:"desc"`
	Content string `gorm:"type:longtext;not null" json:"content"`
	Img     string `gorm:"type:varchar(200)" json:"img"`
}
