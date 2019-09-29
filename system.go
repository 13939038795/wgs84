// Package system implements the wgs84.system interface.
package wgs84

import (
	"math"
)

// system implements the wgs84.system interface.
type system struct {
	toXYZ   func(a, b, c float64, sph spheroid) (x, y, z float64)
	fromXYZ func(x, y, z float64, sph spheroid) (a, b, c float64)
}

// ToXYZ is used in the wgs84.system interface.
func (sys system) ToXYZ(a, b, c float64, s Spheroid) (x, y, z float64) {
	if s == nil {
		s = Datum().Spheroid
	}
	if sys.toXYZ == nil {
		return lonLat().ToXYZ(a, b, c, s)
	}
	sph := spheroid{s.A(), s.Fi()}
	return sys.toXYZ(a, b, c, sph)
}

// FromXYZ is used in the wgs84.system interface.
func (sys system) FromXYZ(x, y, z float64, s Spheroid) (a, b, c float64) {
	if s == nil {
		s = Datum().Spheroid
	}
	if sys.fromXYZ == nil {
		return lonLat().FromXYZ(x, y, z, s)
	}
	sph := spheroid{s.A(), s.Fi()}
	return sys.fromXYZ(x, y, z, sph)
}

func lonLat() system {
	N := func(φ float64, sph spheroid) float64 {
		return sph.A() / math.Sqrt(1-sph.E2()*math.Pow(math.Sin(φ), 2))
	}
	return system{
		toXYZ: func(lon, lat, h float64, sph spheroid) (x, y, z float64) {
			x = (N(radian(lat), sph) + h) * math.Cos(radian(lon)) * math.Cos(radian(lat))
			y = (N(radian(lat), sph) + h) * math.Cos(radian(lat)) * math.Sin(radian(lon))
			z = (N(radian(lat), sph)*math.Pow(sph.A()*(1-sph.F()), 2)/(sph.A2()) + h) * math.Sin(radian(lat))
			return
		},
		fromXYZ: func(x, y, z float64, sph spheroid) (lon, lat, h float64) {
			sd := math.Sqrt(x*x + y*y)
			T := math.Atan(z * sph.A() / (sd * sph.B()))
			B := math.Atan((z + sph.E2()*(sph.A2())/sph.B()*
				math.Pow(math.Sin(T), 3)) / (sd - sph.E2()*sph.A()*math.Pow(math.Cos(T), 3)))
			h = sd/math.Cos(B) - N(B, sph)
			return degree(math.Atan2(y, x)), degree(B), h
		},
	}
}

func transverseMercator(lonf, latf, scale, eastf, northf float64) system {
	M := func(φ float64, sph spheroid) float64 {
		return sph.A() * ((1-sph.E2()/4-3*sph.E4()/64-5*sph.E6()/256)*φ -
			(3*sph.E2()/8+3*sph.E4()/32+45*sph.E6()/1024)*math.Sin(2*φ) +
			(15*sph.E4()/256+45*sph.E6()/1024)*math.Sin(4*φ) -
			(35*sph.E6()/3072)*math.Sin(6*φ))
	}
	N := func(φ float64, sph spheroid) float64 {
		return sph.A() / math.Sqrt(1-sph.E2()*sin2(φ))
	}
	T := tan2
	C := func(φ float64, sph spheroid) float64 {
		return sph.Ei2() * cos2(φ)
	}
	return system{
		toXYZ: func(east, north, h float64, sph spheroid) (x, y, z float64) {
			east -= eastf
			north -= northf
			Mi := M(radian(latf), sph) + north/scale
			μ := Mi / (sph.A() * (1 - sph.E2()/4 - 3*sph.E4()/64 - 5*sph.E6()/256))
			φ1 := μ + (3*sph.Ei()/2-27*sph.Ei3()/32)*math.Sin(2*μ) +
				(21*sph.Ei2()/16-55*sph.Ei4()/32)*math.Sin(4*μ) +
				(151*sph.Ei3()/96)*math.Sin(6*μ) +
				(1097*sph.Ei4()/512)*math.Sin(8*μ)
			R1 := sph.A() * (1 - sph.E2()) / math.Pow(1-sph.E2()*sin2(φ1), 3/2)
			D := east / (N(φ1, sph) * scale)
			φ := φ1 - (N(φ1, sph)*math.Tan(φ1)/R1)*(D*D/2-(5+3*T(φ1)+10*C(φ1, sph)-4*C(φ1, sph)*C(φ1, sph)-9*sph.Ei2())*
				math.Pow(D, 4)/24+(61+90*T(φ1)+298*C(φ1, sph)+45*T(φ1)*T(φ1)-252*sph.Ei2()-3*C(φ1, sph)*C(φ1, sph))*
				math.Pow(D, 6)/720)
			λ := radian(lonf) + (D-(1+2*T(φ1)+C(φ1, sph))*D*D*D/6+(5-2*C(φ1, sph)+
				28*T(φ1)-3*C(φ1, sph)*C(φ1, sph)+8*sph.Ei2()+24*T(φ1)*T(φ1))*
				math.Pow(D, 5)/120)/math.Cos(φ1)
			return lonLat().ToXYZ(degree(λ), degree(φ), h, sph)
		},
		fromXYZ: func(x, y, z float64, sph spheroid) (east, north, h float64) {
			lon, lat, h := lonLat().FromXYZ(x, y, z, sph)
			φ := radian(lat)
			A := (radian(lon) - radian(lonf)) * math.Cos(φ)
			east = scale*N(φ, sph)*(A+(1-T(φ)+C(φ, sph))*
				math.Pow(A, 3)/6+(5-18*T(φ)+T(φ)*T(φ)+72*C(φ, sph)-58*sph.Ei2())*
				math.Pow(A, 5)/120) + eastf
			north = scale*(M(φ, sph)-M(radian(latf), sph)+N(φ, sph)*math.Tan(φ)*
				(A*A/2+(5-T(φ)+9*C(φ, sph)+4*C(φ, sph)*C(φ, sph))*
					math.Pow(A, 4)/24+(61-58*T(φ)+T(φ)*T(φ)+600*C(φ, sph)-330*sph.Ei2())*math.Pow(A, 6)/720)) + northf
			return
		},
	}
}

func utm(zone float64, northern bool) system {
	if northern {
		return transverseMercator(zone*6-183, 0, 0.9996, 500000, 0)
	}
	return transverseMercator(zone*6-183, 0, 0.9996, 500000, 10000000)
}

func gk(zone float64) system {
	return transverseMercator(zone*3, 0, 1, zone*1000000+500000, 0)
}

func mercator(lonf, scale, eastf, northf float64) system {
	return system{
		toXYZ: func(east, north, h float64, sph spheroid) (x, y, z float64) {
			east = (east - eastf) / scale
			north = (north - northf) / scale
			t := math.Exp(-north * sph.A())
			φ := math.Pi/2 - 2*math.Atan(t)
			for i := 0; i < 5; i++ {
				φ = math.Pi/2 - 2*math.Atan(t*math.Pow((1-sph.E()*math.Sin(φ))/(1+sph.E()*math.Sin(φ)), sph.E()/2))
			}
			return lonLat().ToXYZ(east/sph.A()+lonf, degree(φ), h, sph)
		},
		fromXYZ: func(x, y, z float64, sph spheroid) (east, north, h float64) {
			lon, lat, h := lonLat().FromXYZ(x, y, z, sph)
			east = scale * sph.A() * (radian(lon) - radian(lonf))
			north = scale * sph.A() / 2 *
				math.Log(1+math.Sin(radian(lat))/(1-math.Sin(radian(lat)))*
					math.Pow((1-sph.E()*math.Sin(radian(lat)))/(1+sph.E()*math.Sin(radian(lat))), math.E))
			return
		},
	}
}

func webMercator() system {
	return system{
		toXYZ: func(east, north, h float64, sph spheroid) (x, y, z float64) {
			lon := degree(east / sph.A())
			lat := math.Atan(math.Exp(north/sph.A()))*degree(1)*2 - 90
			return lonLat().ToXYZ(lon, lat, h, sph)
		},
		fromXYZ: func(x, y, z float64, sph spheroid) (east, north, h float64) {
			lon, lat, h := lonLat().FromXYZ(x, y, z, sph)
			east = radian(lon) * sph.A()
			north = math.Log(math.Tan(radian((90+lat)/2))) * sph.A()
			return
		},
	}
}

func lambertConformalConic1SP(lonf, latf, scale, eastf, northf float64) system {
	t := func(φ float64, sph spheroid) float64 {
		return math.Tan(math.Pi/4-φ/2) /
			math.Pow((1-sph.E()*math.Sin(φ))/(1+sph.E()*math.Sin(φ)), sph.E()/2)
	}
	m := func(φ float64, sph spheroid) float64 {
		return math.Cos(φ) / math.Sqrt(1-sph.E2()*sin2(φ))
	}
	n := math.Sin(radian(latf))
	F := func(sph spheroid) float64 {
		return m(radian(latf), sph) / (n * math.Pow(t(radian(latf), sph), n))
	}
	ρ := func(φ float64, sph spheroid) float64 {
		return sph.A() * F(sph) * math.Pow(t(φ, sph)*scale, n)
	}
	return system{
		toXYZ: func(east, north, h float64, sph spheroid) (x, y, z float64) {
			ρi := math.Sqrt(math.Pow(east-eastf, 2) + math.Pow(ρ(radian(latf), sph)-(north-northf), 2))
			if n < 0 {
				ρi = -ρi
			}
			ti := math.Pow(ρi/(sph.A()*scale*F(sph)), 1/n)
			φ := math.Pi/2 - 2*math.Atan(ti)
			for i := 0; i < 5; i++ {
				φ = math.Pi/2 - 2*math.Atan(ti*math.Pow((1-sph.E()*math.Sin(φ))/(1+sph.E()*math.Sin(φ)), sph.E()/2))
			}
			λ := math.Atan((east-eastf)/(ρ(radian(latf), sph)-(north-northf)))/n + radian(lonf)
			return lonLat().ToXYZ(degree(λ), degree(φ), h, sph)
		},
		fromXYZ: func(x, y, z float64, sph spheroid) (east, north, h float64) {
			lon, lat, h := lonLat().FromXYZ(x, y, z, sph)
			θ := n * (radian(lon) - radian(lonf))
			east = eastf + ρ(radian(lat), sph)*math.Sin(θ)
			north = northf + ρ(radian(latf), sph) - ρ(radian(lat), sph)*math.Cos(θ)
			return
		},
	}
}

func lambertConformalConic2SP(lonf, latf, lat1, lat2, eastf, northf float64) system {
	t := func(φ float64, sph spheroid) float64 {
		return math.Tan(math.Pi/4-φ/2) /
			math.Pow((1-sph.E()*math.Sin(φ))/(1+sph.E()*math.Sin(φ)), sph.E()/2)
	}
	m := func(φ float64, sph spheroid) float64 {
		return math.Cos(φ) / math.Sqrt(1-sph.E2()*sin2(φ))
	}
	n := func(sph spheroid) float64 {
		if radian(lat1) == radian(lat2) {
			return math.Sin(radian(lat1))
		}
		return (math.Log(m(radian(lat1), sph)) - math.Log(m(radian(lat2), sph))) /
			(math.Log(t(radian(lat1), sph)) - math.Log(t(radian(lat2), sph)))
	}
	F := func(sph spheroid) float64 {
		return m(radian(lat1), sph) / (n(sph) * math.Pow(t(radian(lat1), sph), n(sph)))
	}
	ρ := func(φ float64, sph spheroid) float64 {
		return sph.A() * F(sph) * math.Pow(t(φ, sph), n(sph))
	}
	return system{
		toXYZ: func(east, north, h float64, sph spheroid) (x, y, z float64) {
			ρi := math.Sqrt(math.Pow(east-eastf, 2) + math.Pow(ρ(radian(latf), sph)-(north-northf), 2))
			if n(sph) < 0 {
				ρi = -ρi
			}
			ti := math.Pow(ρi/(sph.A()*F(sph)), 1/n(sph))
			φ := math.Pi/2 - 2*math.Atan(ti)
			for i := 0; i < 5; i++ {
				φ = math.Pi/2 - 2*math.Atan(ti*math.Pow((1-sph.E()*math.Sin(φ))/(1+sph.E()*math.Sin(φ)), sph.E()/2))
			}
			λ := math.Atan((east-eastf)/(ρ(radian(latf), sph)-(north-northf)))/n(sph) + radian(lonf)
			return lonLat().ToXYZ(degree(λ), degree(φ), h, sph)
		},
		fromXYZ: func(x, y, z float64, sph spheroid) (east, north, h float64) {
			lon, lat, h := lonLat().FromXYZ(x, y, z, sph)
			θ := n(sph) * (radian(lon) - radian(lonf))
			east = eastf + ρ(radian(lat), sph)*math.Sin(θ)
			north = northf + ρ(radian(latf), sph) - ρ(radian(lat), sph)*math.Cos(θ)
			return
		},
	}
}

func albersEqualAreaConic(lonf, latf, lat1, lat2, eastf, northf float64) system {
	m := func(φ float64, sph spheroid) float64 {
		return math.Cos(φ) / math.Sqrt(1-sph.E2()*sin2(φ))
	}
	q := func(φ float64, sph spheroid) float64 {
		return (1 - sph.E2()) * (math.Sin(φ)/(1-sph.E2()*sin2(φ)) -
			(1/(2*sph.E()))*math.Log((1-sph.E()*math.Sin(φ))/(1+sph.E()*math.Sin(φ))))
	}
	n := func(sph spheroid) float64 {
		if radian(lat1) == radian(lat2) {
			return math.Sin(radian(lat1))
		}
		return (m(radian(lat1), sph)*m(radian(lat1), sph) - m(radian(lat2), sph)*m(radian(lat2), sph)) /
			(q(radian(lat2), sph) - q(radian(lat1), sph))
	}
	C := func(sph spheroid) float64 {
		return m(radian(lat1), sph)*m(radian(lat1), sph) + n(sph)*q(radian(lat1), sph)
	}
	ρ := func(φ float64, sph spheroid) float64 {
		return sph.A() * math.Sqrt(C(sph)-n(sph)*q(φ, sph)) / n(sph)
	}
	return system{
		toXYZ: func(east, north, h float64, sph spheroid) (x, y, z float64) {
			east -= eastf
			north -= northf
			ρi := math.Sqrt(east*east + math.Pow(ρ(radian(latf), sph)-north, 2))
			qi := (C(sph) - ρi*ρi*n(sph)*n(sph)/sph.A2()) / n(sph)
			φ := math.Asin(qi / 2)
			for i := 0; i < 5; i++ {
				φ += math.Pow(1-sph.E2()*sin2(φ), 2) /
					(2 * math.Cos(φ)) * (qi/(1-sph.E2()) -
					math.Sin(φ)/(1-sph.E2()*sin2(φ)) +
					1/(2*sph.E())*math.Log((1-sph.E()*math.Sin(φ))/(1+sph.E()*math.Sin(φ))))
			}
			θ := math.Atan(east / (ρ(radian(latf), sph) - north))
			return lonLat().ToXYZ(degree(radian(lonf)+θ/n(sph)), degree(φ), h, sph)
		},
		fromXYZ: func(x, y, z float64, sph spheroid) (east, north, h float64) {
			lon, lat, h := lonLat().FromXYZ(x, y, z, sph)
			θ := n(sph) * (radian(lon) - radian(lonf))
			east = eastf + ρ(radian(lat), sph)*math.Sin(θ)
			north = northf + ρ(radian(latf), sph) - ρ(radian(lat), sph)*math.Cos(θ)
			return
		},
	}
}

func equidistantConic(lonf, latf, lat1, lat2, eastf, northf float64) system {
	M := func(φ float64, sph spheroid) float64 {
		return sph.A() * ((1-sph.E2()/4-3*sph.E4()/64-5*sph.E6()/256)*φ -
			(3*sph.E2()/8+3*sph.E4()/32+45*sph.E6()/1024)*math.Sin(2*φ) +
			(15*sph.E4()/256+45*sph.E6()/1024)*math.Sin(4*φ) -
			(35*sph.E6()/3072)*math.Sin(6*φ))
	}
	m := func(φ float64, sph spheroid) float64 {
		return math.Cos(φ) / math.Sqrt(1-sph.E2()*sin2(φ))
	}
	n := func(sph spheroid) float64 {
		if radian(lat1) == radian(lat2) {
			return math.Sin(radian(lat1))
		}
		return sph.A() * (m(radian(lat1), sph) - m(radian(lat2), sph)) / (M(radian(lat2), sph) - M(radian(lat1), sph))
	}
	G := func(sph spheroid) float64 {
		return m(radian(lat1), sph)/n(sph) + M(radian(lat1), sph)/sph.A()
	}
	ρ := func(φ float64, sph spheroid) float64 {
		return sph.A()*G(sph) - M(φ, sph)
	}
	return system{
		toXYZ: func(east, north, h float64, sph spheroid) (x, y, z float64) {
			east -= eastf
			north -= northf
			ρi := math.Sqrt(east*east + math.Pow(ρ(radian(latf), sph)-north, 2))
			if n(sph) < 0 {
				ρi = -ρi
			}
			Mi := sph.A()*G(sph) - ρi
			μ := Mi / (sph.A() * (1 - sph.E2()/4 - 3*sph.E4()/64 - 5*sph.E6()/256))
			φ := μ + (3*sph.Ei()/2-27*sph.Ei3()/32)*math.Sin(2*μ) +
				(21*sph.Ei2()/16-55*sph.Ei4()/32)*math.Sin(4*μ) +
				(151*sph.Ei3()/96)*math.Sin(6*μ) +
				(1097*sph.Ei4()/512)*math.Sin(8*μ)
			θ := math.Atan(east / (ρ(radian(latf), sph) - north))
			return lonLat().ToXYZ(degree((radian(lonf) + θ/n(sph))), degree(φ), h, sph)
		},
		fromXYZ: func(x, y, z float64, sph spheroid) (east, north, h float64) {
			lon, lat, h := lonLat().FromXYZ(x, y, z, sph)
			θ := n(sph) * (radian(lon) - radian(lonf))
			east = eastf + ρ(radian(lat), sph)*math.Sin(θ)
			north = northf + ρ(radian(latf), sph) - ρ(radian(lat), sph)*math.Cos(θ)
			return
		},
	}
}

func sin2(east float64) float64 {
	return math.Pow(math.Sin(east), 2)
}

func cos2(east float64) float64 {
	return math.Pow(math.Cos(east), 2)
}

func tan2(east float64) float64 {
	return math.Pow(math.Tan(east), 2)
}

func degree(r float64) float64 {
	return r * 180 / math.Pi
}

func radian(d float64) float64 {
	return d * math.Pi / 180
}