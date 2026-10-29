"""Optional FastAPI service for expense prediction.

Run:  uvicorn app:app --port 8001
The Go backend calls POST /predict. If this service is down, the main app still works.
"""
from typing import List

from fastapi import FastAPI
from pydantic import BaseModel, Field

from predictor import InsufficientData, predict_next_month

app = FastAPI(title="Expense Prediction Service", version="1.0.0")


class MonthPoint(BaseModel):
    month: str = Field(..., examples=["2026-01"])
    total: float


class PredictRequest(BaseModel):
    history: List[MonthPoint] = Field(..., max_length=120)


@app.get("/health")
def health():
    return {"status": "ok"}


@app.post("/predict")
def predict(req: PredictRequest):
    try:
        return predict_next_month([p.model_dump() for p in req.history])
    except InsufficientData as exc:
        # Not an error: the dashboard simply explains why no prediction is shown.
        return {"available": False, "reason": str(exc)}
