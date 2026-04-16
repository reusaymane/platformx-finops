import pandas as pd
from prophet import Prophet
from datetime import datetime

def train_and_forecast(df: pd.DataFrame, periods: int = 30) -> pd.DataFrame:
    """
    Train a Prophet model on historical cost data and forecast future costs.
    
    Args:
        df: DataFrame with columns 'ds' (date) and 'y' (cost)
        periods: number of days to forecast
    
    Returns:
        DataFrame with ds, yhat, yhat_lower, yhat_upper for forecast period only
    """
    if len(df) < 14:
        raise ValueError(f"Not enough data: {len(df)} days (minimum 14 required)")

    # Prophet expects ds as datetime
    df = df.copy()
    df["ds"] = pd.to_datetime(df["ds"])
    df["y"] = df["y"].astype(float)

    model = Prophet(
        yearly_seasonality=True,
        weekly_seasonality=True,
        daily_seasonality=False,
        changepoint_prior_scale=0.05,  # conservative — avoids overfitting spikes
        interval_width=0.80,           # 80% confidence interval
    )

    # Suppress Prophet's verbose output
    import logging
    logging.getLogger("prophet").setLevel(logging.WARNING)
    logging.getLogger("cmdstanpy").setLevel(logging.WARNING)

    model.fit(df)

    future = model.make_future_dataframe(periods=periods)
    forecast = model.predict(future)

    # Return only future predictions
    last_date = df["ds"].max()
    future_forecast = forecast[forecast["ds"] > last_date][
        ["ds", "yhat", "yhat_lower", "yhat_upper"]
    ].copy()

    # Clamp negative predictions to 0
    future_forecast["yhat"] = future_forecast["yhat"].clip(lower=0)
    future_forecast["yhat_lower"] = future_forecast["yhat_lower"].clip(lower=0)
    future_forecast["yhat_upper"] = future_forecast["yhat_upper"].clip(lower=0)

    return future_forecast

def model_version() -> str:
    return f"prophet-v1-{datetime.now().strftime('%Y%m%d')}"
