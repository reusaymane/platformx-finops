import os
import logging
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import Optional
from datetime import date

from app.db import get_cost_history, get_namespaces, save_forecast
from app.forecaster import train_and_forecast, model_version

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="PlatformX ML Forecasting", version="1.0.0")

class ForecastPoint(BaseModel):
    date: date
    predicted_cost: float
    lower_bound: float
    upper_bound: float

class ForecastResponse(BaseModel):
    model_config = {"protected_namespaces": ()}
    namespace: str
    model_version: str
    forecast_days: int
    monthly_predicted: float
    budget_warning: bool
    forecast: list[ForecastPoint]

@app.get("/healthz")
def health():
    return {"status": "ok"}

@app.get("/namespaces")
def list_namespaces():
    return {"namespaces": get_namespaces()}

@app.get("/forecast/{namespace}", response_model=ForecastResponse)
def forecast(namespace: str, days: int = 30, budget: Optional[float] = None):
    logger.info(f"forecasting costs for namespace={namespace} days={days}")

    df = get_cost_history(namespace, days=365)
    if df.empty:
        raise HTTPException(status_code=404, detail=f"No data for namespace '{namespace}'")

    try:
        forecast_df = train_and_forecast(df, periods=days)
    except ValueError as e:
        raise HTTPException(status_code=422, detail=str(e))

    # Save to TimescaleDB
    mv = model_version()
    save_forecast(namespace, "prod", forecast_df, mv)

    # Build response
    points = [
        ForecastPoint(
            date=row["ds"].date(),
            predicted_cost=round(float(row["yhat"]), 4),
            lower_bound=round(float(row["yhat_lower"]), 4),
            upper_bound=round(float(row["yhat_upper"]), 4),
        )
        for _, row in forecast_df.iterrows()
    ]

    monthly_predicted = sum(p.predicted_cost for p in points[:30])
    budget_warning = budget is not None and monthly_predicted > budget * 0.80

    logger.info(f"forecast complete namespace={namespace} monthly_predicted={monthly_predicted:.2f}")

    return ForecastResponse(
        namespace=namespace,
        model_version=mv,
        forecast_days=len(points),
        monthly_predicted=round(monthly_predicted, 2),
        budget_warning=budget_warning,
        forecast=points,
    )

@app.get("/forecast/{namespace}/summary")
def forecast_summary(namespace: str):
    """Returns a quick summary without full forecast list — for dashboard widgets"""
    df = get_cost_history(namespace, days=90)
    if df.empty:
        raise HTTPException(status_code=404, detail=f"No data for namespace '{namespace}'")

    forecast_df = train_and_forecast(df, periods=30)
    monthly = sum(forecast_df["yhat"].clip(lower=0))

    return {
        "namespace": namespace,
        "monthly_predicted_usd": round(float(monthly), 2),
        "next_7_days_usd": round(float(forecast_df["yhat"].head(7).sum()), 2),
        "trend": "up" if forecast_df["yhat"].iloc[-1] > forecast_df["yhat"].iloc[0] else "down",
    }
