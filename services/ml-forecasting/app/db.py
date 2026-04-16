import os
import pandas as pd
from sqlalchemy import create_engine, text

def get_engine():
    url = os.getenv("DATABASE_URL").replace("postgres://", "postgresql://")
    return create_engine(url)

def get_cost_history(namespace: str, days: int = 365) -> pd.DataFrame:
    engine = get_engine()
    query = text("""
        SELECT
            time_bucket('1 day', time AT TIME ZONE 'UTC') AS ds,
            SUM(cost_usd) AS y
        FROM cost_records
        WHERE namespace = :namespace
          AND time > NOW() - (:days || ' days')::interval
        GROUP BY ds
        ORDER BY ds ASC
    """)
    with engine.connect() as conn:
        df = pd.read_sql(query, conn, params={"namespace": namespace, "days": days})
    df["ds"] = pd.to_datetime(df["ds"]).dt.tz_localize(None)
    return df

def get_namespaces() -> list[str]:
    engine = get_engine()
    with engine.connect() as conn:
        result = conn.execute(text("SELECT DISTINCT namespace FROM cost_records"))
        return [row[0] for row in result]

def save_forecast(namespace: str, environment: str, forecast_df: pd.DataFrame, model_version: str):
    engine = get_engine()
    rows = []
    for _, row in forecast_df.iterrows():
        rows.append({
            "namespace": namespace,
            "environment": environment,
            "forecast_date": row["ds"].date(),
            "predicted_cost": float(row["yhat"]),
            "lower_bound": float(row["yhat_lower"]),
            "upper_bound": float(row["yhat_upper"]),
            "model_version": model_version,
        })
    with engine.connect() as conn:
        conn.execute(text("""
            INSERT INTO forecasts
                (namespace, environment, forecast_date, predicted_cost, lower_bound, upper_bound, model_version)
            VALUES
                (:namespace, :environment, :forecast_date, :predicted_cost, :lower_bound, :upper_bound, :model_version)
            ON CONFLICT DO NOTHING
        """), rows)
        conn.commit()
