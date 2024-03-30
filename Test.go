package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "cloud.google.com/go/spanner"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt"
    "google.golang.org/api/option"
)

// DataDTO represents the Data Transfer Object
type DataDTO struct {
    RefNumber          string    `json:"ref_number"`
    ConsentAccepted    bool      `json:"consent_accepted"`
    DateOfConsent      time.Time `json:"date_of_consent"`
    InbestData         string    `json:"inbest_data"`
}

// DataDAO represents the Data Access Object
type DataDAO struct {
    dbClient *spanner.Client
}

// NewDataDAO creates a new instance of DataDAO
func NewDataDAO(ctx context.Context, dbClient *spanner.Client) *DataDAO {
    return &DataDAO{
        dbClient: dbClient,
    }
}

// SaveData saves data to the database
func (dao *DataDAO) SaveData(ctx context.Context, data *DataDTO) error {
    _, err := dao.dbClient.Apply(ctx, []*spanner.Mutation{
        spanner.InsertOrUpdate("your-table-name", []string{"ref_number", "consent_accepted", "date_of_consent", "inbest_data"}, []interface{}{data.RefNumber, data.ConsentAccepted, data.DateOfConsent, data.InbestData}),
    })
    return err
}

// JWTSecret is the secret key used for JWT signing
var JWTSecret = []byte("your-secret-key")

// authenticate middleware verifies JWT token
func authenticate() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := c.GetHeader("Authorization")
        if tokenString == "" {
            log.Println("Authorization failed: No JWT token provided")
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            return
        }

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return JWTSecret, nil
        })

        if err != nil || !token.Valid {
            log.Println("Authorization failed: Invalid JWT token")
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            return
        }

        c.Next()
    }
}

// generateToken generates JWT token
func generateToken() (string, error) {
    token := jwt.New(jwt.SigningMethodHS256)
    token.Claims = jwt.MapClaims{
        "exp": time.Now().Add(time.Hour * 24).Unix(),
    }
    return token.SignedString(JWTSecret)
}

func main() {
    // Set up Google Cloud Spanner client
    ctx := context.Background()
    projectID := "your-project-id"
    instanceID := "your-spanner-instance-id"
    databaseID := "your-spanner-database-id"
    client, err := spanner.NewClient(ctx, "projects/"+projectID+"/instances/"+instanceID+"/databases/"+databaseID, option.WithCredentialsFile("path/to/your/service-account-key.json"))
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // Create a new Gin router
    router := gin.Default()

    // Initialize DataDAO
    dataDAO := NewDataDAO(ctx, client)

    // Define a POST endpoint to save JSON data to the database
    router.POST("/save", authenticate(), func(c *gin.Context) {
        var data DataDTO
        if err := c.BindJSON(&data); err != nil {
            log.Printf("Failed to bind data: %v", err)
            c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to bind data"})
            return
        }

        // Save data
        err := dataDAO.SaveData(ctx, &data)
        if err != nil {
            log.Printf("Data not saved: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
            return
        }

        c.JSON(http.StatusOK, gin.H{"message": "Data saved successfully"})
    })

    // Define an endpoint to generate JWT token (for testing purpose)
    router.GET("/token", func(c *gin.Context) {
        token, err := generateToken()
        if err != nil {
            log.Printf("Failed to generate token: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
            return
        }
        c.JSON(http.StatusOK, gin.H{"token": token})
    })

    // Run the server
    router.Run(":8080")
}