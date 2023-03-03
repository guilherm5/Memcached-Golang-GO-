package main

import (
	"log"
	"strconv"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB = Init()

var MC = memcache.New("127.0.0.1:11211")

type Users struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func Init() *gorm.DB {
	dsn := "host=localhost user=postgres password=ivaneteJC dbname=memcached port=5432 sslmode=disable "
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	} else {
		log.Println("sucesso ao conectar")
	}
	return db
}

func UsersFunc(c *gin.Context) {
	var user Users
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"erro ao decodificar body para realizar insert": err})
		return
	}

	//adicionando usuario no postgresql
	if err := DB.Create(&user).Error; err != nil {
		c.JSON(500, gin.H{"erro ao realizar insert": err})
		log.Println("erro ao realizar insert", err)
		return
	}

	//adicionando user no memcached
	item := &memcache.Item{
		Key:        "user_" + strconv.Itoa(int(user.ID)),
		Value:      []byte(user.Name),
		Expiration: 10,
	}

	if err := MC.Add(item); err != nil {
		c.JSON(500, gin.H{"erro ao adicionar dado no mccached": err})
		log.Println("erro ao adicionar dado no mccached", err)
		return
	}
	c.JSON(200, user)
}

func UsersFuncID(c *gin.Context) {
	//get por id

	//checando se o parametro name esta no cached
	id := c.Param("id")
	item, err := MC.Get("user_" + id)
	if err == nil {
		c.JSON(200, gin.H{
			"name": string(item.Value),
		})
		log.Println("VIM DO CACHE")
		return
	}

	//get user postgres
	var user Users
	if err := DB.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"error": "Usuario nao existe"})
		} else {
			c.JSON(500, gin.H{"error": err})
		}
		return
	}

	//adicionando item no memcached
	item = &memcache.Item{
		Key:        "user_" + strconv.Itoa(int(user.ID)),
		Value:      []byte(user.Name),
		Expiration: 10,
	}
	if err := MC.Add(item); err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, user)
	log.Println("VIM DO BANCO")
}

func main() {
	Init()
	router := gin.Default()

	router.POST("/users", UsersFunc)
	router.GET("/users/:id", UsersFuncID)

	//conectando ao memcached

	router.Run(":8080")
}
