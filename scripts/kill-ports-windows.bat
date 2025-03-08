@echo off
setlocal EnableDelayedExpansion

echo Starting to close ports 3000 and 5173...
echo.

REM 
set "ports=3000 5173"

REM 循环处理每个端口
for %%p in (%ports%) do (
    echo Checking port %%p...
    
    REM 使用 netstat 查找占用端口的 PID
    for /f "tokens=5" %%i in ('netstat -aon ^| findstr :%%p') do (
        set "pid=%%i"
        echo Found process with PID !pid! using port %%p
        
        REM 强制终止进程
        taskkill /PID !pid! /F
        if !errorlevel! equ 0 (
            echo Successfully closed port %%p ^(PID: !pid!^)
        ) else (
            echo Failed to close port %%p ^(PID: !pid!^)
        )
    )
    
    REM 检查是否还有进程占用该端口
    netstat -aon | findstr :%%p >nul
    if !errorlevel! neq 0 (
        echo Port %%p is now free
    ) else (
        echo Warning: Port %%p might still be in use
    )
    echo.
)

echo.
echo Port closing operation completed.
echo Press any key to exit...
pause >nul