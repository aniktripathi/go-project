"""Expense prediction logic (no web code, so it is easy to test).

Input : monthly expense totals, e.g. [{"month": "2026-01", "total": 12500.0}, ...]
Output: an estimate for next month. It is only an estimate, never a guarantee.
"""
import numpy as np
import pandas as pd
from sklearn.linear_model import LinearRegression

MIN_MONTHS = 4          # fewer months than this is not enough to say anything useful
TREND_MIN_MONTHS = 6    # a trend line needs a little more history
TREND_MIN_R2 = 0.30     # if the line fits poorly, a simple average is more honest

DISCLAIMER = ("This is a statistical estimate based on your past spending. "
              "It is not a guarantee of what you will actually spend.")


class InsufficientData(ValueError):
    """Raised when there is not enough clean data to make a prediction."""


def clean_history(history):
    """Turn raw input into a continuous monthly series. Bad rows are ignored."""
    rows = []
    for item in history or []:
        try:
            month = pd.to_datetime(str(item["month"]), format="%Y-%m")
            total = float(item["total"])
        except (KeyError, TypeError, ValueError):
            continue                      # missing or malformed entry
        if not np.isfinite(total) or total < 0:
            continue                      # NaN, infinity or negative totals are invalid
        rows.append((month.to_period("M"), total))

    if not rows:
        raise InsufficientData("No valid monthly data was provided.")

    series = pd.Series(dict_sum(rows)).sort_index()
    # Months with no expenses at all inside the range count as 0 spending.
    full_index = pd.period_range(series.index.min(), series.index.max(), freq="M")
    return series.reindex(full_index, fill_value=0.0)


def dict_sum(rows):
    """Sum totals when the same month appears more than once."""
    out = {}
    for period, total in rows:
        out[period] = out.get(period, 0.0) + total
    return out


def predict_next_month(history):
    series = clean_history(history)
    n = len(series)
    if n < MIN_MONTHS:
        raise InsufficientData(
            f"At least {MIN_MONTHS} months of expense history are needed (found {n}).")

    y = series.to_numpy(dtype=float)
    x = np.arange(n).reshape(-1, 1)

    method = "moving_average"
    prediction = float(np.mean(y[-3:]))
    residuals = y[-3:] - prediction

    if n >= TREND_MIN_MONTHS:
        model = LinearRegression().fit(x, y)
        r2 = float(model.score(x, y))
        if r2 >= TREND_MIN_R2:
            method = "linear_regression"
            prediction = float(model.predict([[n]])[0])
            residuals = y - model.predict(x)

    prediction = max(prediction, 0.0)            # spending cannot be negative
    margin = float(np.std(residuals, ddof=1)) if len(residuals) > 1 else 0.0

    next_month = series.index[-1] + 1
    return {
        "available": True,
        "predicted_total": round(prediction, 2),
        "range_low": round(max(prediction - margin, 0.0), 2),
        "range_high": round(prediction + margin, 2),
        "target_month": str(next_month),
        "method": method,
        "months_used": int(n),
        "disclaimer": DISCLAIMER,
    }
