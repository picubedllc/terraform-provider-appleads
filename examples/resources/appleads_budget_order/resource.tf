# Manages an Apple Ads budget order (LOC / agency billing container).
# Campaigns reference the managed order via budget_orders.

resource "appleads_budget_order" "q1" {
  name                = "Q1 2026 LOC — Screenshot Organizer"
  budget_amount       = "5000.00"
  budget_currency     = "USD"
  start_date          = "2026-01-01T00:00:00.000"
  end_date            = "2026-03-31T23:59:59.999"
  primary_buyer_name  = "Example Agency Buyer"
  primary_buyer_email = "buyer@example.com"
  billing_email       = "billing@example.com"
  client_name         = "Example Client"
  order_number        = "PO-2026-Q1-001"
}

resource "appleads_campaign" "search" {
  name                  = "Search — Screenshot Organizer (example)"
  adam_id               = data.appleads_app.app.adam_id
  countries_or_regions  = ["US"]
  status                = "PAUSED"
  daily_budget_amount   = "25.00"
  daily_budget_currency = "USD"
  budget_orders         = [appleads_budget_order.q1.id]

  lifecycle {
    prevent_destroy = true
  }
}
