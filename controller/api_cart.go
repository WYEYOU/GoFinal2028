package controller

import (
	"gofinal/model"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CartController(router *gin.Engine) {
	routers := router.Group("/cart")
	{
		routers.GET("/product", products)
		routers.GET("/products", searchProducts) // ค้นหาสินค้า
		routers.POST("/add", addToCart)          // เพิ่มสินค้าลงรถเข็น
		routers.GET("/all", getCartsByCustomerID)
	}

}

func products(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
	}
	var country []model.Product
	if err := DB.Find(&country).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": country})
}

func searchProducts(c *gin.Context) {
	var products []model.Product
	filters := map[string]interface{}{}

	// รับค่าจาก Query Parameters
	description := strings.TrimSpace(c.Query("description"))
	minPriceStr := c.Query("min_price")
	maxPriceStr := c.Query("max_price")

	// ตรวจสอบว่ามี description หรือไม่
	if description != "" {
		filters["description LIKE ?"] = "%" + description + "%"
	}

	// แปลง minPrice และ maxPrice เป็น float64
	var minPrice, maxPrice float64
	var err error

	if minPriceStr != "" {
		minPrice, err = strconv.ParseFloat(minPriceStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid min_price"})
			return
		}
		filters["price >= ?"] = minPrice
	}

	if maxPriceStr != "" {
		maxPrice, err = strconv.ParseFloat(maxPriceStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid max_price"})
			return
		}
		filters["price <= ?"] = maxPrice
	}

	// Query ข้อมูลโดยใช้ Dynamic Filtering
	query := DB.Model(&model.Product{})
	for key, value := range filters {
		query = query.Where(key, value)
	}

	// ค้นหาข้อมูล
	if err := query.Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": products})
}

// เพิ่มสินค้าลงในรถเข็น
func addToCart(c *gin.Context) {
	var request struct {
		CustomerID int    `json:"customer_id"`
		CartName   string `json:"cart_name"`
		ProductID  int    `json:"product_id"`
		Quantity   int    `json:"quantity"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var cart model.Cart
	// ตรวจสอบว่ามีรถเข็นชื่อนี้อยู่หรือไม่
	if err := DB.Where("customer_id = ? AND cart_name = ?", request.CustomerID, request.CartName).First(&cart).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// ถ้าไม่มีให้สร้างรถเข็นใหม่
			cart = model.Cart{
				CustomerID: request.CustomerID,
				CartName:   request.CartName,
			}
			DB.Create(&cart)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// ตรวจสอบว่าสินค้านี้อยู่ในรถเข็นแล้วหรือไม่
	var cartItem model.CartItem
	if err := DB.Where("cart_id = ? AND product_id = ?", cart.CartID, request.ProductID).First(&cartItem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// ถ้าไม่มีให้เพิ่มสินค้าลงตะกร้า
			cartItem = model.CartItem{
				CartID:    cart.CartID,
				ProductID: request.ProductID,
				Quantity:  request.Quantity,
			}
			DB.Create(&cartItem)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// ถ้ามีสินค้าอยู่แล้ว เพิ่มจำนวนเข้าไป
		DB.Model(&cartItem).Update("quantity", cartItem.Quantity+request.Quantity)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product added to cart successfully"})
}

func getCartsByCustomerID(c *gin.Context) {
	customerID := c.Query("customer_id")

	if customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id is required"})
		return
	}

	var carts []model.Cart
	if err := DB.Preload("Items").Preload("Items.Product").
		Where("customer_id = ?", customerID).
		Find(&carts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": carts})
}
