#!/usr/bin/env python3
"""สกัดยอดรวมจาก const DATA ใน dashboard เดิม มาเป็น fixture สำหรับ golden test

ตัวเลขในไฟล์ผลลัพธ์คือ "ความจริง" ที่ dashboard เดิมแสดงอยู่ ระบบใหม่ต้องคำนวณได้ตรงกันทุกตัว
รันซ้ำได้ ผลลัพธ์คงที่เสมอ เพราะเรียงคีย์ทุกระดับก่อนเขียน
"""
import json, sys, collections

SRC = sys.argv[1]
OUT = sys.argv[2]

with open(SRC, encoding="utf8") as f:
    for line in f:
        if line.lstrip().startswith("const DATA"):
            data = json.loads(line[line.index("{"): line.rindex("}") + 1])
            break
    else:
        sys.exit("ไม่พบ const DATA ในไฟล์")

def agg(rows, key_fn, val_idx):
    out = collections.defaultdict(float)
    for r in rows:
        out[key_fn(r)] += r[val_idx] or 0
    return {k: round(v, 4) for k, v in sorted(out.items())}

def ym(r, y=0, m=1):
    return f"{r[y]}-{r[m]:02d}"

si, so, st = data["sellin"], data["sellout"], data["stock"]
tg, tr, cu, pr = data["target"], data["tracking"], data["customers"], data["products"]

golden = {
    "_source": "dashboard_sellin_Final.html · const DATA",
    "counts": {
        "customers": len(cu), "products": len(pr), "sellin": len(si),
        "sellout": len(so), "stock": len(st), "target": len(tg), "tracking": len(tr),
    },
    "customers": {
        "active": sum(1 for c in cu if c["status"] == "Active"),
        "codes": sorted(c["code"] for c in cu),
        "province_count": len({p for c in cu if c["status"] == "Active" for p in c["provinces"]}),
    },
    "sellin": {
        # index: 0=year 1=month 2=day 3=dc 4=group 5=category 6=doc 7=cartons 8=revenue
        "total_revenue": round(sum(r[8] or 0 for r in si), 4),
        "total_cartons": round(sum(r[7] or 0 for r in si), 4),
        "unique_docs": len({r[6] for r in si}),
        "by_year_month": agg(si, ym, 8),
        "by_dc": agg(si, lambda r: r[3], 8),
        "by_group_cartons": agg(si, lambda r: r[4], 7),
        "by_category_cartons": agg(si, lambda r: r[5], 7),
    },
    "sellout": {
        # index: 0=year 1=month 2=dc 3=prod 4=desc 5=group 6=category 7=cartons 8=revenue
        "total_revenue": round(sum(r[8] or 0 for r in so), 4),
        "total_cartons": round(sum(r[7] or 0 for r in so), 4),
        "by_year_month": agg(so, ym, 8),
        "by_dc": agg(so, lambda r: r[2], 8),
        "by_group_cartons": agg(so, lambda r: r[5], 7),
    },
    "stock": {
        # index: 0=year 1=month 2=dc 3=prod 4=desc 5=group 6=category 7=beg 8=in 9=out 10=end
        "total_beginning": round(sum(r[7] or 0 for r in st), 4),
        "total_sell_in": round(sum(r[8] or 0 for r in st), 4),
        "total_sell_out": round(sum(r[9] or 0 for r in st), 4),
        "total_ending": round(sum(r[10] or 0 for r in st), 4),
        "ending_by_year_month": agg(st, ym, 10),
    },
    "target": {
        "total": round(sum(r[4] or 0 for r in tg), 4),
        "by_year_month": agg(tg, ym, 4),
    },
    "tracking": {
        "total_target_qty": round(sum(r[4] or 0 for r in tr), 4),
        "total_cartons": round(sum(r[5] or 0 for r in tr), 4),
    },
}

with open(OUT, "w", encoding="utf8") as f:
    json.dump(golden, f, ensure_ascii=False, indent=2, sort_keys=True)
    f.write("\n")

print(f"เขียน {OUT}")
print(f"  sell-in  {golden['counts']['sellin']:>6,} แถว  revenue {golden['sellin']['total_revenue']:>18,.2f}")
print(f"  sell-out {golden['counts']['sellout']:>6,} แถว  revenue {golden['sellout']['total_revenue']:>18,.2f}")
print(f"  stock    {golden['counts']['stock']:>6,} แถว  ending  {golden['stock']['total_ending']:>18,.2f}")
print(f"  target   {golden['counts']['target']:>6,} แถว  total   {golden['target']['total']:>18,.2f}")
