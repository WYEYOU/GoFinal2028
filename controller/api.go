package controller

import (
	"gofinal/model"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var DB *gorm.DB

func SetDB(db *gorm.DB) {
	DB = db
}

func CustomerController(router *gin.Engine) {
	routers := router.Group("/customers")
	{
		routers.GET("/", customers)
		routers.POST("/login", loginCustomer)      // ลูกค้าล็อกอิน
		routers.GET("/:id", getCustomer)           // ดึงข้อมูลลูกค้า
		routers.PUT("/:id/address", updateAddress) // แก้ไขที่อยู่ลูกค้า
		routers.POST("/reg", registerCustomer)     // สมัคร
	}

}

func customers(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
	}
	var country []model.Customer
	if err := DB.Find(&country).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": country})
}

// ฟังก์ชันเข้ารหัสรหัสผ่าน
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// สมัครใช้งาน
func registerCustomer(c *gin.Context) {
	var input struct {
		FirstName   string `json:"first_name" binding:"required"`
		LastName    string `json:"last_name" binding:"required"`
		Email       string `json:"email" binding:"required"`
		PhoneNumber string `json:"phone_number"`
		Address     string `json:"address"`
		Password    string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// เข้ารหัสรหัสผ่าน
	hashedPassword, err := hashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
		return
	}

	customer := model.Customer{
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		Email:       input.Email,
		PhoneNumber: input.PhoneNumber,
		Address:     input.Address,
		Password:    hashedPassword, // บันทึกเป็นรหัสผ่านที่เข้ารหัสแล้ว
	}

	if err := DB.Create(&customer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Customer registered successfully"})
}

// ฟังก์ชันตรวจสอบรหัสผ่าน
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ฟังก์ชันล็อกอิน
func loginCustomer(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var customer model.Customer
	if err := DB.Where("email = ?", input.Email).First(&customer).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if !checkPasswordHash(input.Password, customer.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// คืนค่าข้อมูลลูกค้า ยกเว้นรหัสผ่าน
	c.JSON(http.StatusOK, gin.H{
		"customer_id":  customer.CustomerID,
		"first_name":   customer.FirstName,
		"last_name":    customer.LastName,
		"email":        customer.Email,
		"phone_number": customer.PhoneNumber,
		"address":      customer.Address,
	})
}

// ฟังก์ชันดึงข้อมูลลูกค้า
func getCustomer(c *gin.Context) {
	customerID := c.Param("id")
	var customer model.Customer

	if err := DB.First(&customer, customerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	// คืนค่าข้อมูลลูกค้า โดยเรียงลำดับตามโครงสร้างของตาราง และไม่แสดงรหัสผ่าน
	c.JSON(http.StatusOK, gin.H{
		"customer_id":  customer.CustomerID,
		"first_name":   customer.FirstName,
		"last_name":    customer.LastName,
		"email":        customer.Email,
		"phone_number": customer.PhoneNumber,
		"address":      customer.Address,
		"created_at":   customer.CreatedAt,
		"updated_at":   customer.UpdatedAt,
	})
}

// ฟังก์ชันอัปเดตที่อยู่ลูกค้า
func updateAddress(c *gin.Context) {
	customerID := c.Param("id")
	var input struct {
		Address string `json:"address" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := DB.Model(&model.Customer{}).Where("customer_id = ?", customerID).Update("address", input.Address).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address updated successfully"})
}
