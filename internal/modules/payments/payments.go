package payments

import (
 "math"
 "math/big"
 "strconv"
 "strings"
 "github.com/gofiber/fiber/v3"
)
type Service struct{}
func NewService()*Service{return &Service{}}
func(s *Service)Routes(app fiber.Router){
 r:=app.Group("/payments")
 r.Post("/",func(c fiber.Ctx)error{return c.Status(201).SendString("This action adds a new payment")})
 r.Get("/",func(c fiber.Ctx)error{return c.SendString("This action returns all payments")})
 r.Get("/:id",func(c fiber.Ctx)error{return c.SendString("This action returns a #"+legacyNumericID(c.Params("id"))+" payment")})
 r.Patch("/:id",func(c fiber.Ctx)error{return c.SendString("This action updates a #"+legacyNumericID(c.Params("id"))+" payment")})
 r.Delete("/:id",func(c fiber.Ctx)error{return c.SendString("This action removes a #"+legacyNumericID(c.Params("id"))+" payment")})
}
// Match JavaScript Number coercion used by NestJS unary +, including radix prefixes.
func legacyNumericID(raw string)string{
 s:=strings.TrimSpace(raw)
 if s=="" {return "0"}
 if s=="Infinity" || s=="+Infinity"{return "Infinity"}
 if s=="-Infinity"{return "-Infinity"}
 if len(s)>2 && s[0]=='0' {
  base:=0
  switch s[1] {case 'x','X':base=16;case 'b','B':base=2;case 'o','O':base=8}
  if base!=0 {
   n,ok:=new(big.Int).SetString(s[2:],base);if !ok{return "NaN"}
   f,_:=new(big.Float).SetInt(n).Float64()
   return formatJSNumber(f)
  }
 }
 n,err:=strconv.ParseFloat(s,64);if err!=nil{return "NaN"}
 return formatJSNumber(n)
}
func formatJSNumber(n float64)string{
 if math.IsNaN(n){return "NaN"}
 if math.IsInf(n,1){return "Infinity"}
 if math.IsInf(n,-1){return "-Infinity"}
 if n==0{return "0"}
 abs:=math.Abs(n)
 if abs>=1e-6 && abs<1e21 {return strconv.FormatFloat(n,'f',-1,64)}
 out:=strconv.FormatFloat(n,'e',-1,64)
 parts:=strings.SplitN(out,"e",2)
 if len(parts)!=2{return out}
 exp,e:=strconv.Atoi(parts[1]);if e!=nil{return out}
 if exp>=0{return parts[0]+"e+"+strconv.Itoa(exp)}
 return parts[0]+"e"+strconv.Itoa(exp)
}
