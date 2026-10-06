package catalog

type Product struct {
	ID      string `json:"id"`
	Charges int64  `json:"charges"`
	Price   string `json:"price"`
}

var products = []Product{
	{ID: "charges_100", Charges: 100, Price: "€4.99"},
	{ID: "charges_500", Charges: 500, Price: "€19.99"},
	{ID: "charges_1200", Charges: 1200, Price: "€39.99"},
}

func All() []Product {
	out := make([]Product, len(products))
	copy(out, products)
	return out
}

func Lookup(id string) (Product, bool) {
	for _, p := range products {
		if p.ID == id {
			return p, true
		}
	}
	return Product{}, false
}
