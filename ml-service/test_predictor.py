"""Run with:  python -m unittest -v      (no extra packages needed besides requirements.txt)"""
import unittest

from predictor import InsufficientData, predict_next_month


def series(values, start_year=2026, start_month=1):
    out = []
    y, m = start_year, start_month
    for v in values:
        out.append({"month": f"{y}-{m:02d}", "total": v})
        m += 1
        if m > 12:
            m, y = 1, y + 1
    return out


class PredictorTests(unittest.TestCase):
    def test_clear_upward_trend_uses_regression(self):
        r = predict_next_month(series([100, 200, 300, 400, 500, 600]))
        self.assertEqual(r["method"], "linear_regression")
        self.assertAlmostEqual(r["predicted_total"], 700.0, places=1)
        self.assertEqual(r["target_month"], "2026-07")
        self.assertIn("estimate", r["disclaimer"])

    def test_short_history_uses_moving_average(self):
        r = predict_next_month(series([100, 200, 300, 400]))
        self.assertEqual(r["method"], "moving_average")
        self.assertAlmostEqual(r["predicted_total"], 300.0)   # mean of 200, 300, 400

    def test_noisy_flat_history_falls_back_to_average(self):
        r = predict_next_month(series([500, 300, 520, 310, 505, 295]))
        self.assertEqual(r["method"], "moving_average")

    def test_too_little_data(self):
        with self.assertRaises(InsufficientData):
            predict_next_month(series([100, 200, 300]))
        with self.assertRaises(InsufficientData):
            predict_next_month([])
        with self.assertRaises(InsufficientData):
            predict_next_month(None)

    def test_bad_rows_are_ignored(self):
        data = series([100, 200, 300, 400, 500])
        data += [{"month": "garbage", "total": 5}, {"month": "2027-01"}, {"total": 3},
                 {"month": "2026-08", "total": -50}, {"month": "2026-09", "total": float("nan")}]
        r = predict_next_month(data)
        self.assertEqual(r["months_used"], 5)

    def test_prediction_is_never_negative(self):
        r = predict_next_month(series([600, 500, 400, 300, 200, 100]))
        self.assertGreaterEqual(r["predicted_total"], 0.0)
        self.assertGreaterEqual(r["range_low"], 0.0)

    def test_year_rollover(self):
        r = predict_next_month(series([100, 110, 120, 130], start_year=2026, start_month=9))
        self.assertEqual(r["target_month"], "2027-01")

    def test_missing_month_counts_as_zero_and_duplicates_are_summed(self):
        data = [{"month": "2026-01", "total": 100}, {"month": "2026-01", "total": 100},
                {"month": "2026-02", "total": 200}, {"month": "2026-04", "total": 400},
                {"month": "2026-05", "total": 500}]
        r = predict_next_month(data)
        self.assertEqual(r["months_used"], 5)   # Jan..May with March filled as 0


if __name__ == "__main__":
    unittest.main()
