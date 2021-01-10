![gotham](https://user-images.githubusercontent.com/39353278/103892416-99f6c880-50fc-11eb-8869-af197ca81fd1.png)

## Repos and frameworks used

- Dependency Injection Container ( https://github.com/sarulabs/di )
- Handler ( https://echo.labstack.com )
- ORM (https://gorm.io)
- Validation (https://github.com/go-ozzo/ozzo-validation)
- Cron (https://github.com/jasonlvhit/gocron)

## Table of contents

![GitHub](https://img.shields.io/github/license/tolgaOzen/go-gotham)
![GitHub top language](https://img.shields.io/github/languages/top/tolgaozen/go-gotham)
![GitHub last commit](https://img.shields.io/github/last-commit/tolgaozen/go-gotham)

- [Architecture](#architecture)
    * [Di Container](#di-container)
    * [Services](#services)
    * [Provider](#provider)
    * [Defs](#defs)
    - [Di Scopes](#di-scopes)
        * [App Scope](#app-scope)
        * [Request Scope](#request-scope)
        * [Unscoped](#unscoped)
    * [Controllers](#controllers)
    - [Middlewares](#conditional-middlewares):
        * [Create](#create)
        * [Conditional Middlewares](#conditional-middlewares)
- [Database](#database)
    * [Supported Databases](#supported-databases)
    * [Procedures](#procedures)
    * [Db Scopes](#db-scopes)
    * [Migrations](#migrations)
- [ORM](#orm)
- [Requests](#requests)
    * [Create Requests](#create-new-requests)
    * [Bind Request And Validate](#bind-request-and-validate)
    * [More Info For Validations Rules](#more-info-for-validations-rules)
    * [Custom Rules](#custom-rules)
- [Auth](#auth)
    * [JWT](#jwt)
- [Jobs](#jobs)
- [Features To Be Added Soon](#features-to-be-added-soon)

## Setup

You can start using this repository by cloning it.

# Architecture

## Di Container

![architecture](https://user-images.githubusercontent.com/39353278/104106317-d6c1eb80-52c5-11eb-8fd7-1206ec5dd12d.png)

## Services

### Example

/services/database.go
```go
type DatabaseService struct {
	DbConfig config.Database
}

type DatabaseConnecter interface {
	open() gorm.Dialector
}

func NewDatabaseService(dbConfig config.Database) *DatabaseService {
	return &DatabaseService{
		DbConfig: dbConfig,
	}
}

func (s DatabaseService) OpenDatabase() (db gorm.Dialector) {
	var d DatabaseConnecter
	switch s.DbConfig.DbConnection {
	case "postgres":
		d = Postgres{s}
	case "mysql":
		d = Mysql{s}
	default:
		d = Mysql{s}
	}
	db = d.open()
	return
}

func (DatabaseService) ConnectDatabase(dialector gorm.Dialector) (db *gorm.DB, err error) {
	return gorm.Open(dialector, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
}

// Mysql
type Mysql struct{
	DatabaseService
}

func (m Mysql) open() (dia gorm.Dialector) {
	dsn := m.DbConfig.DbUserName + ":" + m.DbConfig.DbPassword + "@(" + m.DbConfig.DbHost + ")/" + m.DbConfig.DbDatabase + "?charset=utf8&parseTime=True&loc=Local"
	return mysql.Open(dsn)
}

// Postgresql
type Postgres struct{
	DatabaseService
}

func (p Postgres) open() (dia gorm.Dialector) {
	return postgres.New(postgres.Config{
		DSN:                  "user=" + p.DbConfig.DbUserName + " host=" + p.DbConfig.DbHost + " password=" + p.DbConfig.DbPassword + " dbname=" + p.DbConfig.DbDatabase + " port=" + p.DbConfig.DbPort + " sslmode=disable",
		PreferSimpleProtocol: true,
	})
}
```

## Provider

You will have to write the service definitions and register them in a Provider.

/app/provider/appServiceProvider.go
```go
func (p *Provider) Load() error {
  
    if err := p.AddDefSlice(defs.DatabaseServiceDefs); err != nil {
       return err
    }

    if err := p.AddDefSlice(defs.CustomService1Defs); err != nil {
       return err
    }

    if err := p.AddDefSlice(defs.CustomService2Defs); err != nil {
       return err
    }

    return nil
}
```

### Defs

A Definitions consists of parts where we write the dependencies required to create the object and where we can determine the life cycles of objects.

#### Example

/app/defs/database.go
```go
var DatabaseServiceDefs = []dingo.Def{
	{
		Name:  "db-pool",
		Scope: di.App,
		Build: func() (gorm.Dialector, error) {
			return services.NewDatabaseService(config.GetDbConfig()).OpenDatabase(), nil
		},
	},
	{
		Name:  "db",
		Scope: di.Request,
		Build: func(dia gorm.Dialector) (db *gorm.DB,err error) {
			return services.DatabaseService{}.ConnectDatabase(dia)
		},
		Params: dingo.Params{
			"0": dingo.Service("db-pool"),
		},
		Close: func(db *gorm.DB) error {
			sqlDB, _ := db.DB()
			return sqlDB.Close()
		},
	},
}
```

Like the example above, the db object is dependent on the dp-pool object. While calling the db object, the db-pool object is injected into the db object, and the  db object is created.

### Di Scopes
Scopes allow us to control the life cycle of the created objects.


#### App Scope
App scope is the widest scope. It is created once during the application's run time.

The db-pool object in the example above is an example.

#### Request Scope
The request scope is a sub-scope. Container can generate children in the next scope thanks to the SubContainer method.

The container creates a subcontainer and adds the request context via DicSubContainerSetterMiddleware.

So how can request scope objects be accessed?
#### Example

```
dic.Db(c.Request())
```

When the request is finished, request scope objects are cleaned from the container.


#### Unscoped

app can retrieve a request-object with unscoped methods.

```
db := app.Application.Container.UnscopedGetDb()

var user models.User
db.Find(&user)

app.Application.Container.Clean()
```

Once the objects created with unscoped methods are no longer used,
you can call the Clean method. In this case, the Close function will be called on the object.

### Controllers

#### Create

Creating a file in the controllers folder

#### Examples

```go
func IsVerified(next echo.HandlerFunc) echo.HandlerFunc {
    return func (c echo.Context) error {
        u := c.Get("user").(*jwt.Token)
        claims := u.Claims.(*config.JwtCustomClaims)

        user := models.User{}
        if err := dic.Db(c.Request()).First(&user, claims.Id).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                return false, echo.ErrUnauthorized
            }
           return c.JSON(echo.ErrInternalServerError, err)
        }

        if user.IsVerified() {
            return next(c)
        }

        return c.JSON(http.StatusBadRequest, helpers.ErrorResponse(http.StatusBadRequest, "your email not verified"))
    }
}
```

```go
r.GET("/users/:user", controllers.UserController{}.Show, GMiddleware.IsVerified, GMiddleware.IsAdmin)
```

## Middlewares

### Create

Creating a file in the middleware folder

### Examples

```go
func IsVerified(next echo.HandlerFunc) echo.HandlerFunc {
    return func (c echo.Context) error {
        u := c.Get("user").(*jwt.Token)
        claims := u.Claims.(*config.JwtCustomClaims)

        user := models.User{}
        if err := dic.Db(c.Request()).First(&user, claims.Id).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                return false, echo.ErrUnauthorized
            }
           return c.JSON(echo.ErrInternalServerError, err)
        }

        if user.IsVerified() {
            return next(c)
        }

        return c.JSON(http.StatusBadRequest, helpers.ErrorResponse(http.StatusBadRequest, "your email not verified"))
    }
}
```

```go
r.GET("/users/:user", controllers.UserController{}.Show, GMiddleware.IsVerified, GMiddleware.IsAdmin)
```

### Conditional Middlewares

#### Examples

/middlewares/isAdmin.go

```go
type IsAdmin struct {}

func (i IsAdmin) control(c echo.Context) (bool bool, err error) {
    u := c.Get("user").(*jwt.Token)
    claims := u.Claims.(*config.JwtCustomClaims)

    user := models.User{}

    if err := dic.Db(c.Request()).First(&user, claims.Id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return false, echo.ErrUnauthorized
        }
        return false, echo.ErrInternalServerError
    }

    if user.IsAdmin() {
       return true, nil
    }

    return false, errors.New("you are not admin")
}

```

/middlewares/isVerified.go
```go
type IsVerified struct{}

func (i IsVerified) control(c echo.Context) (bool bool, err error) {
    u := c.Get("user").(*jwt.Token)
    claims := u.Claims.(*config.JwtCustomClaims)

    user := models.User{}
    if err := dic.Db(c.Request()).First(&user, claims.Id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return false, echo.ErrNotFound
        }
        return false, echo.ErrInternalServerError
    }

    if user.IsVerified() {
        return true, nil
    }

    return false, errors.New("your email not verified")
}
```

#### OR

```go
r.GET("/users/:user", controllers.UserController{}.Show, GMiddleware.Or([]GMiddleware.MiddlewareI{GMiddleware.IsAdmin{}, GMiddleware.IsVerified{}}))
```

Authenticated user must be admin or verified

#### AND

```go
r.GET("/users/:user", controllers.UserController{}.Show, GMiddleware.And([]GMiddleware.MiddlewareI{GMiddleware.IsAdmin{}, GMiddleware.IsVerified{}}))
```

Authenticated user must be admin and verified

## Database

### Supported Databases

supports databases MySQL and PostgreSQL

### Procedures

Creating a file in the models/procedures folder, create a type and create(db *gorm.DB), drop(db *gorm.DB), dropIfExist(
db *gorm.DB) methods for that type. create getter function for this procedure

You can look at the example below

#### Example

models/procedures/getUsersCount.go

```go
type UserCount struct {
    Count int  `json:"rate"`
}

func (UserCount) create(db *gorm.DB) error {
    sql := `CREATE PROCEDURE GetUsersCount()
        BEGIN
          SELECT COUNT(*) as count FROM users;
        END`

    return db.Exec(sql).Error
}

func (UserCount) drop(db *gorm.DB) error {
    sql := `DROP PROCEDURE GetUserCount;`
    return db.Exec(sql).Error
}

func (UserCount) dropIfExist(db *gorm.DB) error {
    sql := `DROP PROCEDURE IF EXISTS GetUserCount;`
    return db.Exec(sql).Error
}

func GetUserCount(db *gorm.DB) UserCount {
    var returnVal UserCount
    db.Raw("CALL GetUserCount()").Scan(&returnVal)
    return returnVal
}
```

#### Register Procedure

models/procedures/base.go

```go
func Initialize() {
    db := app.Application.Container.UnscopedGetDb()

    // UserCount Register
    _ = DropProcedureIfExist(UserCount{}, db)
    _ = CreateProcedure(UserCount{}, db)
    
    app.Application.Container.Clean()
}
```

### Db Scopes

#### Pagination Scope

In Controller Usage

controllers/userController.go index method

```go
request := new(requests.Pagination)

if err = c.Bind(request); err != nil {
    return
}

var count int64
dic.Db(c.Request()).Model(&models.User{}).Count(&count)

var users []models.User

if err := dic.Db(c.Request()).Scopes(scopes.Paginate(request, models.User{}, "name")).Find(&users).Error; err != nil {
    return echo.ErrInternalServerError
}

return c.JSON(http.StatusOK, helpers.SuccessResponse(accessories.Paginator{
    TotalRecord: int(count),
    Records:     users,
    Limit:       request.Limit,
    Page:        request.Page,
}))
```

You can add pagination to any request object

```go
type ExampleRequest struct {
    validation.Validatable `json:"-" form:"-" query:"-"`

    /**
    * PAGINATION
    */
    Pagination Pagination

    /**
    * BODY
    */
    Verified int `json:"verified" form:"verified" query:"verified"`
}
```

In Controller Usage

```go
if err := dic.Db(c.Request()).Scopes(scopes.Paginate(&request.Pagination, models.User{}, "name")).Find(&users).Error; err != nil {
    return echo.ErrInternalServerError
}
```

### Migrations

When you create a model, insert it into the Initialize() function of the database/migration/base.go.

#### Register Migration

models/procedures/base.go

#### Example

```go
func Initialize() {
    db := app.Application.Container.UnscopedGetDb()

    _ = db.AutoMigrate(&models.User{})
    
    app.Application.Container.Clean()
}
```

## ORM

Check out fantastic gorm library https://gorm.io/docs/

## Requests

### Create New Requests

Creating a file in the requests folder, create a type and create a Validate() method for that type. You can look at the
examples below

### Bind Request And Validate

#### Example

Request Object

```go
type LoginRequest struct {
    validation.Validatable `json:"-" form:"-" query:"-"`
 
    /**
    * BODY
    */
    Email    string `json:"email" form:"email" query:"email"`
    Password string `json:"password" form:"password" query:"password"`
}

func (r LoginRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Email, validation.Required, validation.Length(4, 50), is.Email),
        validation.Field(&r.Password, validation.Required, validation.Length(8, 50)),
    )
}
```

In Controller Usage

```go
request := new(requests.LoginRequest)

if err = c.Bind(request); err != nil {
    return
}

v := request.Validate()

if v != nil {
    return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
         "errors": v,
    })
}

// you can access binded request object
fmt.println(request.Email)
```

### More Info For Validations Rules

Check out ozzo-validation library https://github.com/go-ozzo/ozzo-validation

### Custom Rules

#### Examples

rules/stringEquals.go

```go
func StringEquals(str string) validation.RuleFunc {
return func (value interface{}) error {
    s, _ := value.(string)

    if s != str {
       return errors.New("unexpected string")
    }

    return nil
    }
}
```

#### Usage In Any Request Object Validate Method

```go
func (r ExampleRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Name, validation.By(rules.StringEquals("john"))),
    )
}
```

## Auth

### JWT

#### Config

/config/jwt.go
```go
type JwtCustomClaims struct {
    Id               uint   `json:"id"`
    Name             string `json:"name"`
    Email            string `json:"email"`
    jwt.StandardClaims
}
```

#### Middleware

/routers/api.go
```go
r := e.Group("/restricted")

c := middleware.JWTConfig{
	Claims:     &config.JwtCustomClaims{},
    SigningKey: []byte(app.Application.Config.SecretKey),
}

r.Use(middleware.JWTWithConfig(c))
```

#### LoginController

/controllers/loginController.go
```go
exp := time.Now().Add(time.Minute * 15).Unix()

claims := &config.JwtCustomClaims{
    Id:    user.ID,
    Name:  user.Name,
    Email: user.Email,
    StandardClaims: jwt.StandardClaims{
        ExpiresAt: exp,
    },
}

token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

t, err := token.SignedString([]byte(app.Application.Config.SecretKey))
if err != nil {
	return
}

data := map[string]interface{}{
    "access_token":      t,
    "access_token_exp":  exp,
    "user":              user,
}

return c.JSON(http.StatusOK, helpers.SuccessResponse(data))
```
In any controlls or middleware you can find information about who owns the token

```go
u := c.Get("user").(*jwt.Token)
claims := u.Claims.(*config.JwtCustomClaims)
```

## Jobs
Check out GoCron https://github.com/jasonlvhit/gocron

## Features To Be Added Soon

- Database seeder
- Unit testing

## Author

> Tolga Özen

> mtolgaozen@gmail.com

## License

MIT License

Copyright (c) 2021 Tolga Özen

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
documentation files (the "Software"), to deal in the Software without restriction, including without limitation the
rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit
persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the
Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.