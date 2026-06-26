#!/usr/bin/env python3
"""
Workalendar Adapter Microservice
Exposes Workalendar library as a REST API for the Semantic Engine
API Port: 8000
"""

from flask import Flask, jsonify, request
from workalendar.usa import UnitedStates
from workalendar.europe import UnitedKingdom, France, Germany, Spain
from workalendar.asia import Japan, China
from workalendar.oceania import Australia

import logging
from datetime import datetime

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# ============================================================================
# Workalendar Registry (Maps region codes to calendar classes)
# ============================================================================

CALENDARS = {
    'US': UnitedStates(),
    'GB': UnitedKingdom(),
    'FR': France(),
    'DE': Germany(),
    'ES': Spain(),
    'JP': Japan(),
    'CN': China(),
    'AU': Australia(),
}

# ============================================================================
# API Endpoints
# ============================================================================

@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({"status": "healthy", "service": "workalendar-adapter"}), 200


@app.route('/holidays', methods=['GET'])
def get_holidays():
    """
    GET /holidays?region=US&year=2026
    Returns holidays for a specific region and year
    """
    region = request.args.get('region', '').upper()
    year = request.args.get('year', type=int)

    if not region or not year:
        return jsonify({"error": "Missing region or year parameter"}), 400

    if region not in CALENDARS:
        return jsonify({"error": f"Unsupported region: {region}"}), 400

    try:
        calendar = CALENDARS[region]
        holidays = calendar.holidays(year)
        
        result = {
            "region": region,
            "year": year,
            "holidays": [
                {
                    "date": str(date),
                    "name": name
                }
                for date, name in holidays
            ]
        }
        
        logger.info(f"Fetched {len(holidays)} holidays for {region}/{year}")
        return jsonify(result), 200
    
    except Exception as e:
        logger.error(f"Error fetching holidays: {e}")
        return jsonify({"error": str(e)}), 500


@app.route('/supported-regions', methods=['GET'])
def supported_regions():
    """Returns list of supported regions"""
    return jsonify({"regions": list(CALENDARS.keys())}), 200


@app.route('/is-holiday', methods=['GET'])
def is_holiday():
    """
    GET /is-holiday?region=US&date=2026-07-04
    Check if a specific date is a holiday
    """
    region = request.args.get('region', '').upper()
    date_str = request.args.get('date', '')

    if not region or not date_str:
        return jsonify({"error": "Missing region or date parameter"}), 400

    if region not in CALENDARS:
        return jsonify({"error": f"Unsupported region: {region}"}), 400

    try:
        date = datetime.strptime(date_str, "%Y-%m-%d").date()
        calendar = CALENDARS[region]
        
        holidays = dict(calendar.holidays(date.year))
        is_holiday = date in holidays
        
        return jsonify({
            "region": region,
            "date": date_str,
            "is_holiday": is_holiday,
            "holiday_name": holidays.get(date)
        }), 200
    
    except ValueError as e:
        return jsonify({"error": f"Invalid date format: {str(e)}"}), 400
    except Exception as e:
        logger.error(f"Error checking holiday: {e}")
        return jsonify({"error": str(e)}), 500


if __name__ == '__main__':
    logger.info("Starting Workalendar Adapter on port 8000")
    app.run(host='0.0.0.0', port=8000, debug=False)
