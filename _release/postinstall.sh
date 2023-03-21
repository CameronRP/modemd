#!/bin/bash
systemctl daemon-reload
systemctl enable modemd.service
systemctl restart modemd.service
