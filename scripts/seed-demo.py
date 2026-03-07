#!/usr/bin/env python3
import json
import sys
from datetime import datetime, timedelta, timezone
from urllib import error, request


BASE_URL = "http://localhost:8080/api/v1"
EMAIL = "demo@budget.local"
PASSWORD = "Demo123!Pass"
NAME = "Demo User"
BASE_CURRENCY = "SGD"


def http(method, path, data=None, token=None, extra_headers=None):
    url = f"{BASE_URL}{path}"
    body = None
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if extra_headers:
        headers.update(extra_headers)
    if data is not None:
        body = json.dumps(data).encode("utf-8")

    req = request.Request(url=url, method=method, headers=headers, data=body)
    try:
        with request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8")
            return json.loads(raw) if raw else {}
    except error.HTTPError as e:
        raw = e.read().decode("utf-8") if e.fp else ""
        raise RuntimeError(f"{method} {path} failed ({e.code}): {raw}")


def auth():
    try:
        resp = http(
            "POST",
            "/auth/register",
            {
                "email": EMAIL,
                "password": PASSWORD,
                "name": NAME,
                "baseCurrency": BASE_CURRENCY,
            },
        )
        return resp["accessToken"]
    except RuntimeError:
        resp = http("POST", "/auth/login", {"email": EMAIL, "password": PASSWORD})
        return resp["accessToken"]


def ensure_category(token, name, cat_type):
    existing = http("GET", "/categories", token=token).get("categories", [])
    for c in existing:
        if c.get("name") == name and c.get("type") == cat_type:
            return c["id"]

    created = http(
        "POST",
        "/categories",
        {"name": name, "type": cat_type},
        token=token,
    )
    return created["category"]["id"]


def get_default_source_asset_id(token):
    assets = http("GET", "/assets", token=token).get("assets", [])
    for asset in assets:
        if not asset.get("isLiability"):
            return asset.get("id")
    if assets:
        return assets[0].get("id")
    return None


def seed_transactions(token, category_ids, source_asset_id):
    tx = http("GET", "/transactions", token=token).get("transactions", [])
    if any((t.get("description") or "").startswith("DEMO:") for t in tx):
        return

    now = datetime.now(timezone.utc)
    entries = [
        (category_ids["salary"], 6500, "DEMO: Monthly salary", now - timedelta(days=27)),
        (category_ids["freelance"], 1200, "DEMO: Freelance project", now - timedelta(days=14)),
        (category_ids["housing"], 1800, "DEMO: Rent payment", now - timedelta(days=26)),
        (category_ids["groceries"], 320, "DEMO: Weekly groceries", now - timedelta(days=20)),
        (category_ids["groceries"], 290, "DEMO: Weekly groceries", now - timedelta(days=13)),
        (category_ids["transport"], 120, "DEMO: Public transport", now - timedelta(days=18)),
        (category_ids["transport"], 95, "DEMO: Ride share", now - timedelta(days=9)),
        (category_ids["entertainment"], 210, "DEMO: Movies and dining", now - timedelta(days=11)),
        (category_ids["travel"], 680, "DEMO: Flight booking", now - timedelta(days=5)),
    ]

    for category_id, amount, desc, tx_date in entries:
        http(
            "POST",
            "/transactions",
            {
                "categoryId": category_id,
                "amount": {"amount": f"{amount:.2f}", "currency": BASE_CURRENCY},
                "transactionDate": tx_date.isoformat(),
                "description": desc,
                "tags": ["demo"],
            },
            token=token,
            extra_headers={"Grpc-Metadata-source-asset-id": source_asset_id},
        )


def seed_budget(token, category_id):
    budgets = http("GET", "/budgets", token=token).get("budgets", [])
    if any(b.get("categoryId") == category_id for b in budgets):
        return

    start = datetime.now(timezone.utc).replace(day=1, hour=0, minute=0, second=0, microsecond=0)
    http(
        "POST",
        "/budgets",
        {
            "categoryId": category_id,
            "amount": {"amount": "1200.00", "currency": BASE_CURRENCY},
            "periodType": "PERIOD_TYPE_MONTHLY",
            "startDate": start.isoformat(),
        },
        token=token,
    )


def pick_asset_type(token, names):
    asset_types = http("GET", "/asset-types", token=token).get("assetTypes", [])
    lowered = {a.get("name", "").lower(): a.get("id") for a in asset_types}
    for n in names:
        if n.lower() in lowered:
            return lowered[n.lower()]
    if asset_types:
        return asset_types[0].get("id")
    raise RuntimeError("No asset types found")


def seed_assets(token):
    assets = http("GET", "/assets", token=token).get("assets", [])
    existing_names = {a.get("name") for a in assets}

    cash_type = pick_asset_type(token, ["Cash", "Checking Account", "Savings Account"])
    inv_type = pick_asset_type(token, ["Stocks", "Investment Account", "Mutual Funds"])
    liability_type = pick_asset_type(token, ["Credit Card", "Loan", "Mortgage"])

    to_create = [
        ("DBS Savings", cash_type, "18500.00", False),
        ("Brokerage Portfolio", inv_type, "42200.00", False),
        ("Credit Card Balance", liability_type, "2300.00", True),
    ]

    for name, asset_type_id, value, is_liability in to_create:
        if name in existing_names:
            continue
        http(
            "POST",
            "/assets",
            {
                "assetTypeId": asset_type_id,
                "name": name,
                "currency": BASE_CURRENCY,
                "currentValue": value,
                "isLiability": is_liability,
            },
            token=token,
        )


def seed_goals(token):
    progress = http("GET", "/goals/progress", token=token).get("progress", [])
    existing_names = {p.get("goal", {}).get("name") for p in progress}

    goals = [
        ("Japan Trip", "6000.00", 90, "2100.00"),
        ("Emergency Fund", "15000.00", 240, "7800.00"),
    ]

    for name, target, days_out, current in goals:
        if name in existing_names:
            continue

        deadline = (datetime.now(timezone.utc) + timedelta(days=days_out)).isoformat()
        created = http(
            "POST",
            "/goals",
            {
                "name": name,
                "targetAmount": {"amount": target, "currency": BASE_CURRENCY},
                "deadline": deadline,
                "notes": "Demo goal",
            },
            token=token,
        )

        goal_id = created.get("goal", {}).get("id")
        if goal_id:
            http(
                "PUT",
                f"/goals/{goal_id}/progress",
                {"currentAmount": {"amount": current, "currency": BASE_CURRENCY}},
                token=token,
            )


def main():
    token = auth()

    category_ids = {
        "salary": ensure_category(token, "Salary", "TRANSACTION_TYPE_INCOME"),
        "freelance": ensure_category(token, "Freelance", "TRANSACTION_TYPE_INCOME"),
        "housing": ensure_category(token, "Housing", "TRANSACTION_TYPE_EXPENSE"),
        "groceries": ensure_category(token, "Groceries", "TRANSACTION_TYPE_EXPENSE"),
        "transport": ensure_category(token, "Transport", "TRANSACTION_TYPE_EXPENSE"),
        "entertainment": ensure_category(token, "Entertainment", "TRANSACTION_TYPE_EXPENSE"),
        "travel": ensure_category(token, "Travel", "TRANSACTION_TYPE_EXPENSE"),
    }

    seed_assets(token)
    source_asset_id = get_default_source_asset_id(token)
    if not source_asset_id:
        raise RuntimeError("No asset found for demo transaction source")

    seed_transactions(token, category_ids, source_asset_id)
    seed_budget(token, category_ids["groceries"])
    seed_goals(token)

    print("Demo seed complete.")
    print(f"Username (email): {EMAIL}")
    print(f"Password: {PASSWORD}")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"Error: {exc}", file=sys.stderr)
        sys.exit(1)
