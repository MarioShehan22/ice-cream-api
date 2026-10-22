package database

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"icecream-api/internal/models"
)

// Seed is idempotent: it only inserts what is missing.
func Seed(db *gorm.DB, bcryptCost int) error {
	users := []struct {
		Name, Email, Password string
		Role                  models.Role
	}{
		{"Olivia Owner", "owner@icecream.test", "Owner@123", models.RoleOwner},
		{"Mark Manager", "manager@icecream.test", "Manager@123", models.RoleManager},
		{"Cathy Cashier", "cashier@icecream.test", "Cashier@123", models.RoleCashier},
	}
	for _, u := range users {
		var count int64
		if err := db.Model(&models.User{}).Where("email = ?", u.Email).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcryptCost)
		if err != nil {
			return err
		}
		if err := db.Create(&models.User{Name: u.Name, Email: u.Email, PasswordHash: string(hash), Role: u.Role}).Error; err != nil {
			return err
		}
	}

	ids := map[string]uint{}
	for _, name := range []string{"Chocolate", "Vanilla", "Strawberry"} {
		var cat models.Category
		if err := db.Where(models.Category{Name: name}).FirstOrCreate(&cat).Error; err != nil {
			return err
		}
		ids[name] = cat.ID
	}

	var itemCount int64
	if err := db.Model(&models.Item{}).Count(&itemCount).Error; err != nil {
		return err
	}
	if itemCount == 0 {
		items := []models.Item{
			{Name: "Double Chocolate Scoop", Description: "Rich dark chocolate", CategoryID: ids["Chocolate"], Price: 2.50, Quantity: 40},
			{Name: "Choco Fudge Sundae", Description: "Chocolate with fudge sauce", CategoryID: ids["Chocolate"], Price: 4.75, Quantity: 25},
			{Name: "Classic Vanilla Scoop", Description: "Madagascar vanilla", CategoryID: ids["Vanilla"], Price: 2.25, Quantity: 60},
			{Name: "Vanilla Bean Cone", Description: "Vanilla in a waffle cone", CategoryID: ids["Vanilla"], Price: 3.25, Quantity: 35},
			{Name: "Strawberry Swirl Scoop", Description: "Fresh strawberry swirl", CategoryID: ids["Strawberry"], Price: 2.50, Quantity: 30},
			{Name: "Strawberry Milkshake", Description: "Thick strawberry shake", CategoryID: ids["Strawberry"], Price: 5.00, Quantity: 20},
		}
		if err := db.Create(&items).Error; err != nil {
			return err
		}
	}

	var custCount int64
	if err := db.Model(&models.Customer{}).Count(&custCount).Error; err != nil {
		return err
	}
	if custCount == 0 {
		customers := []models.Customer{
			{Name: "Nimal Perera", Email: "nimal@example.com", Phone: "0771234567", Address: "12 Beach Rd, Negombo"},
			{Name: "Kasuni Silva", Email: "kasuni@example.com", Phone: "0712345678", Address: "45 Lake Dr, Kandy"},
			{Name: "Ruwan Fernando", Email: "ruwan@example.com", Phone: "0753456789", Address: "7 Hill St, Colombo"},
		}
		if err := db.Create(&customers).Error; err != nil {
			return err
		}
	}
	return nil
}
