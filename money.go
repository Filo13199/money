package money

import (
	"errors"
	"math"
	"math/big"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ZeroMoney, _ = primitive.ParseDecimal128("0")

func Add(m, m1 primitive.Decimal128) (primitive.Decimal128, error) {
	c, e, err := m.BigInt()
	if err != nil {
		return ZeroMoney, err
	}

	c1, e1, err := m1.BigInt()
	if err != nil {
		return ZeroMoney, err
	}
	if e < -323 || e > 308 || e1 < -323 || e1 > 308 {
		return ZeroMoney, errors.New("invalid")
	}
	if e == e1 {
		bgRes := big.NewInt(c.Int64() + c1.Int64())
		val, ok := primitive.ParseDecimal128FromBigInt(bgRes, e)
		if !ok {
			return ZeroMoney, nil
		}
		return val, nil
	} else if e < e1 {
		factor := math.Pow10(e1 - e)
		c1 = c1.Mul(c1, big.NewInt(int64(factor)))
		bgRes := c1.Add(c1, c)
		val, _ := primitive.ParseDecimal128FromBigInt(bgRes, e)
		return val, nil
	}

	// based on trial, any number greater than 10^30, will result in overflow in big.Int type,
	// int64 can represent up to 10^18 without overflow
	factor := bigIntPow10(e - e1)
	c = c.Mul(c, factor)
	bgRes := c.Add(c, c1)
	val, _ := primitive.ParseDecimal128FromBigInt(bgRes, e1)
	return val, nil
}

func Sub(m, m1 primitive.Decimal128) (primitive.Decimal128, error) {
	c, e, err := m.BigInt()
	if err != nil {
		return ZeroMoney, err
	}

	c1, e1, err := m1.BigInt()
	if err != nil {
		return ZeroMoney, err
	}
	if e < -323 || e > 308 || e1 < -323 || e1 > 308 {
		return ZeroMoney, errors.New("invalid")
	}
	c1.Mul(c1, big.NewInt(-1))
	if e == e1 {
		bgRes := big.NewInt(c.Int64() + c1.Int64())
		val, _ := primitive.ParseDecimal128FromBigInt(bgRes, e)
		return val, nil
	} else if e < e1 {
		factor := bigIntPow10(e1 - e)

		c1 = c1.Mul(c1, factor)
		bgRes := c1.Add(c1, c)
		val, _ := primitive.ParseDecimal128FromBigInt(bgRes, e)
		return val, nil
	}

	factor := bigIntPow10(e - e1)

	c = c.Mul(c, factor)
	bgRes := c.Add(c, c1)
	val, _ := primitive.ParseDecimal128FromBigInt(bgRes, e1)
	return val, nil
}

func Mul(m, m1 primitive.Decimal128) (primitive.Decimal128, error) {
	c, e, err := m.BigInt()
	if err != nil {
		return ZeroMoney, err
	}

	c1, e1, err := m1.BigInt()
	if err != nil {
		return ZeroMoney, err
	}

	cMul := c.Mul(c, c1)
	eSum := e + e1
	res, _ := primitive.ParseDecimal128FromBigInt(cMul, eSum)
	return res, nil
}

func SystemRound(roundTo primitive.Decimal128, roundDir string, val primitive.Decimal128) primitive.Decimal128 {
	if roundTo == ZeroMoney {
		return val
	}
	c, e, _ := val.BigInt()
	rtc, rte, _ := roundTo.BigInt()
	if e > rte {
		factor := bigIntPow10(e - rte)
		c = c.Mul(c, factor)
		e = rte
	}

	factor := bigIntPow10(rte - e)

	rtc = rtc.Mul(rtc, factor)
	rem := new(big.Int).Set(c).Mod(c, rtc)
	roundedVal := c
	if rem.Cmp(big.NewInt(0)) != 0 {
		roundedVal = roundedVal.Sub(roundedVal, rem)
		if roundDir == "up" {
			roundedVal.Add(roundedVal, rtc)
		}
	}
	roundedVal.Float64()
	roundedRes, _ := primitive.ParseDecimal128FromBigInt(roundedVal, e)
	return roundedRes
}

func Compare(x, y primitive.Decimal128) (int, error) {
	xc, xe, err := x.BigInt()
	if err != nil {
		return 0, err
	}

	yc, ye, err := y.BigInt()
	if err != nil {
		return 0, err
	}

	xcSign := xc.Sign()
	ycSign := yc.Sign()

	// if m is +ve and m1 is -ve, or vice versa
	if xcSign != ycSign {
		if xcSign < ycSign {
			return -1, nil
		}
		return 1, nil
	}
	// otherwise both numbers have same sign

	// if both numbers have same exponent
	if xe == ye {
		return xc.Cmp(yc), nil
	}

	flipRes := false
	if xcSign < 0 {
		flipRes = true
		xc = xc.Abs(xc)
		yc = yc.Abs(yc)
	}

	//
	xquo, xrem := xc, big.NewInt(0)
	yquo, yrem := yc, big.NewInt(0)

	// 630.523E4 => 630523,1 => 6305230,0
	// 630.5234 => 6305234 -4
	// 6305234/10*(+4) => 630,5234

	if xe > 0 {
		factor := bigIntPow10(xe)
		xquo.Mul(xquo, factor)
	} else {
		factor := bigIntPow10(-xe)
		xquo, xrem = new(big.Int).DivMod(xc, factor, new(big.Int))
	}

	if ye > 0 {
		factor := bigIntPow10(ye)

		yquo.Mul(yquo, factor)
	} else {
		factor := bigIntPow10(-ye)
		yquo, yrem = new(big.Int).DivMod(yc, factor, new(big.Int))
	}

	// 630.5230005 => 6305230005, -7
	// 630.7 => 6307,-1 => 6307*10(-1-(-7)) => 6307000000

	if xe < ye {
		eDiff := ye - xe
		factor := bigIntPow10(eDiff)

		yrem = new(big.Int).Mul(yrem, factor)
	} else if ye < xe {
		eDiff := xe - ye
		factor := bigIntPow10(eDiff)
		xrem.Mul(xrem, factor)
	}

	cmpRes := 0

	// 0.6996969

	if qcmpRes := xquo.Cmp(yquo); qcmpRes == 1 {
		cmpRes = 1
	} else if qcmpRes == -1 {
		cmpRes = -1
	} else {
		if dcmpRes := xrem.Cmp(yrem); dcmpRes == 1 {
			cmpRes = 1
		} else if dcmpRes == -1 {
			cmpRes = -1
		}
	}

	if flipRes {
		cmpRes *= -1
	}

	return cmpRes, nil
}

func bigIntPow10(power int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(power)), big.NewInt(0))
}
