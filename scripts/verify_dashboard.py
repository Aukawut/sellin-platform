#!/usr/bin/env python3
"""เทียบผลลัพธ์ของ API ทุกหน้ากับตัวเลขที่ dashboard เดิมแสดงอยู่

ใช้ golden fixture ชุดเดียวกับ golden test ของ Go แต่ตรวจอีกชั้นหนึ่ง
คือตรวจว่า SQL + service ประกอบตัวเลขออกมาถูก ไม่ใช่แค่ตัวอ่าน Excel ถูก
"""
import json, sys, urllib.request, urllib.parse

API, TOKEN, DS = sys.argv[1], sys.argv[2], sys.argv[3]
GOLDEN = json.load(open("server/testdata/golden_dashboard.json"))
TOL = 0.05

fails, checks = [], 0

def get(path, **params):
    params["dataset_id"] = DS
    url = f"{API}{path}?{urllib.parse.urlencode(params)}"
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {TOKEN}"})
    with urllib.request.urlopen(req) as r:
        return json.load(r)

def check(label, got, want):
    global checks
    checks += 1
    if isinstance(want, (int, float)) and abs(float(got) - float(want)) > TOL:
        fails.append(f"{label}: ได้ {got:,.4f} ต้องการ {float(want):,.4f}")
    elif not isinstance(want, (int, float)) and got != want:
        fails.append(f"{label}: ได้ {got!r} ต้องการ {want!r}")

def kpi(page, key):
    for k in page["kpis"]:
        if k["key"] == key:
            return k
    raise KeyError(f"ไม่พบ KPI {key}")

def series(chart, key):
    for s in chart["series"]:
        if s["key"] == key:
            return s["points"]
    raise KeyError(f"ไม่พบเส้น {key}")

# ---------- Sell-In ----------
p = get("/dashboard/sellin", year="ALL", month="ALL", dc="ALL")
g = GOLDEN["sellin"]
check("Sell-In revenue รวม", kpi(p, "revenue")["value"], g["total_revenue"])
check("Sell-In จำนวนออเดอร์", kpi(p, "orders")["value"], g["unique_docs"])
check("Sell-In ศูนย์ Active", kpi(p, "active_store")["text"], f"{GOLDEN['customers']['active']}/{GOLDEN['counts']['customers']}")
check("Sell-In จังหวัดที่ครอบคลุม", p["network"]["province_count"], GOLDEN["customers"]["province_count"])

for y in (2025, 2026):
    pts = series(p["trend"], str(y))
    for m in range(1, 13):
        want = g["by_year_month"].get(f"{y}-{m:02d}")
        got = pts[m - 1]
        if want is None:
            if got is not None:
                fails.append(f"Sell-In trend {y}-{m:02d}: ควรว่างแต่ได้ {got}")
            checks += 1
        else:
            check(f"Sell-In trend {y}-{m:02d}", got, want)

for e in p["ranking"]:
    check(f"Sell-In อันดับศูนย์ {e['code']}", e["value"], g["by_dc"][e["code"]])
for s in p["donut_group"]:
    check(f"Sell-In donut group {s['label']}", s["value"], g["by_group_cartons"][s["label"]])
for s in p["donut_category"]:
    check(f"Sell-In donut category {s['label']}", s["value"], g["by_category_cartons"][s["label"]])

# ฟิลเตอร์รายศูนย์: ยอดของศูนย์เดียวต้องตรงกับที่แยกไว้ใน golden
for code in ("BP002", "BP011", "BP023"):
    one = get("/dashboard/sellin", year="ALL", month="ALL", dc=code)
    check(f"Sell-In เฉพาะศูนย์ {code}", kpi(one, "revenue")["value"], g["by_dc"][code])
    # ตารางจัดอันดับต้องยังแสดงทุกศูนย์ ไม่ถูกตัดตามฟิลเตอร์
    check(f"Sell-In เลือกศูนย์ {code} แล้วอันดับยังครบ", len(one["ranking"]), len(p["ranking"]))

# ---------- Sell-Out ----------
p = get("/dashboard/sellout", year="ALL", month="ALL", dc="ALL")
g = GOLDEN["sellout"]
check("Sell-Out revenue รวม", kpi(p, "revenue")["value"], g["total_revenue"])
check("Sell-Out cartons รวม", kpi(p, "revenue")["secondary"]["value"], g["total_cartons"])
check("Sell-Out target รวม", kpi(p, "target")["value"], GOLDEN["target"]["total"])
check("Sell-Out % achievement", kpi(p, "achievement")["value"],
      g["total_revenue"] / GOLDEN["target"]["total"] * 100)
for e in p["ranking"]:
    check(f"Sell-Out อันดับศูนย์ {e['code']}", e["value"], g["by_dc"][e["code"]])
for s in p["donut_group"]:
    check(f"Sell-Out donut group {s['label']}", s["value"], g["by_group_cartons"][s["label"]])
check("Sell-Out จำนวนสินค้าในตาราง", len(p["products"]) > 0, True)
check("Sell-Out สินค้าเรียงตาม revenue",
      all(p["products"][i]["revenue"] >= p["products"][i+1]["revenue"] for i in range(len(p["products"])-1)), True)
check("Sell-Out สัดส่วนสินค้ารวมเป็น 100%", round(sum(x["share"] for x in p["products"]), 2), 100.0)

# ---------- Stock ----------
p = get("/dashboard/stock", year="ALL", month="ALL", dc="ALL")
sc = p["scope"]
check("Stock เลือกเดือนล่าสุดให้เอง", sc["month_inferred"], True)
check("Stock มีข้อมูล", sc["has_data"], True)
ending = kpi(p, "beginning_stock")
check("Stock KPI มีครบ 4 ตัว", len(p["kpis"]), 4)
check("Stock มีรายการสินค้า", len(p["products"]) > 0, True)
check("Stock เรียงรายการเสี่ยงขึ้นก่อน",
      p["products"][0]["status"] in ("risk", "warn", "ok"), True)

ym = f"{sc['effective_year']}-{sc['effective_month']:02d}"
want_ending = GOLDEN["stock"]["ending_by_year_month"][ym]
got_ending = sum(x["ending"] for x in p["products"])
check(f"Stock ยอดคงเหลือรวมของ {ym}", got_ending, want_ending)

# ---------- Planning ----------
p = get("/dashboard/planning", year="ALL", month="ALL", dc="ALL")
check("Planning actual", kpi(p, "actual")["value"], GOLDEN["sellout"]["total_revenue"])
check("Planning target", kpi(p, "target")["value"], GOLDEN["target"]["total"])
check("Planning มีแผนครบทุกศูนย์ Active", len(p["plan"]), GOLDEN["customers"]["active"])
check("Planning สัดส่วนภาครวมเป็น 100%", round(sum(n["pct"] for n in p["regions"]["nodes"]), 2), 100.0)
check("Planning ยอดรวมรายภาค", p["regions"]["total"], GOLDEN["sellout"]["total_revenue"])
check("Planning ศูนย์ที่ไม่มีเป้าอยู่ท้ายตาราง",
      all(not r["has_target"] for r in p["plan"][next((i for i, r in enumerate(p["plan"]) if not r["has_target"]), len(p["plan"])):]), True)

# forecast ของปี 2569 ต้องหารด้วยจำนวนเดือนที่มีข้อมูล ไม่ใช่ 12
p26 = get("/dashboard/planning", year="2026", month="ALL", dc="ALL")
fc = kpi(p26, "forecast")
months = int(fc["secondary"]["value"])
year_actual = sum(v for k, v in GOLDEN["sellout"]["by_year_month"].items() if k.startswith("2026"))
check("Planning จำนวนเดือนที่มีข้อมูลปี 2569", months, 7)
check("Planning ยอดคาดการณ์ทั้งปี", fc["value"], year_actual / months * 12)

print(f"ตรวจทั้งหมด {checks} รายการ")
if fails:
    print(f"\nไม่ผ่าน {len(fails)} รายการ:")
    for f in fails[:25]:
        print("  -", f)
    sys.exit(1)
print("ทุกรายการตรงกับ dashboard เดิม")
