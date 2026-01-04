package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

// ================== Data Structures ==================

// ReturnRequest represents input for total return calculator
type ReturnRequest struct {
	InitialInvestment float64
	FinalValue        float64
	Years             float64
}

// ReturnResult represents output of total return calculator
type ReturnResult struct {
	TotalReturn      float64
	PercentReturn    float64
	AnnualizedReturn float64
}

// LoanComparisonRequest represents input for loan vs investing calculator
type LoanComparisonRequest struct {
	LoanAmount           float64
	InterestRate         float64 // Annual rate as percentage (e.g., 5.5 for 5.5%)
	TermMonths           int
	InvestmentReturnRate float64 // Annual rate as percentage
}

// AmortizationRow represents one month in the amortization schedule
type AmortizationRow struct {
	Month            int
	Payment          float64
	Principal        float64
	Interest         float64
	RemainingBalance float64
	InvestmentValue  float64
}

// LoanComparisonResult represents output of loan comparison calculator
type LoanComparisonResult struct {
	MonthlyPayment   float64
	TotalPaid        float64
	TotalInterest    float64
	FinalInvestValue float64
	Difference       float64
	Schedule         []AmortizationRow
}

// InvestmentRequest represents input for recurring investment calculator
type InvestmentRequest struct {
	Ticker    string
	StartDate time.Time
	EndDate   time.Time
	Amount    float64
	Frequency string // "weekly" or "monthly"
}

// InvestmentResult represents output of investment calculator
type InvestmentResult struct {
	Ticker        string
	StartDate     string
	EndDate       string
	FinalValue    float64
	TotalInvested float64
	TotalReturn   float64
	PercentReturn float64
	TotalShares   float64
	Investments   int
}

// StockPrice represents a single day's price
type StockPrice struct {
	Date     time.Time
	AdjClose float64
}

// SavingsGoalRequest represents input for savings goal calculator
type SavingsGoalRequest struct {
	GoalAmount     float64
	CurrentAge     int
	TargetAge      int
	CurrentSavings float64
	AnnualReturn   float64 // percentage
}

// SavingsGoalResult represents output of savings goal calculator
type SavingsGoalResult struct {
	MonthlySavings       float64
	WeeklySavings        float64
	YearlySavings        float64
	YearsToGoal          int
	TotalContributions   float64
	InterestEarned       float64
	CurrentSavingsGrowth float64
}

// FIRERequest represents input for FIRE calculator
type FIRERequest struct {
	CurrentAge     int
	AnnualIncome   float64
	AnnualExpenses float64
	CurrentSavings float64
	AnnualReturn   float64 // percentage
}

// FIREResult represents output of FIRE calculator
type FIREResult struct {
	FIRENumber      float64
	YearsToFIRE     float64
	FIREAge         int
	SavingsRate     float64
	AnnualSavings   float64
	MonthlySavings  float64
	PortfolioAtFIRE float64
}

// InflationRequest represents input for inflation calculator
type InflationRequest struct {
	Amount        float64
	Years         int
	InflationRate float64 // percentage
}

// InflationResult represents output of inflation calculator
type InflationResult struct {
	CurrentAmount       float64
	Years               int
	FutureValueNeeded   float64
	PurchasingPowerLost float64
	PercentLost         float64
}

// DebtRequest represents input for debt payoff calculator
type DebtRequest struct {
	DebtAmount     float64
	InterestRate   float64 // annual percentage
	MinimumPayment float64
	ExtraPayment   float64
}

// DebtResult represents output of debt payoff calculator
type DebtResult struct {
	MinimumMonths   int
	MinimumYears    float64
	MinimumInterest float64
	MinimumTotal    float64
	ExtraPayment    float64
	ExtraMonths     int
	ExtraYears      float64
	ExtraInterest   float64
	ExtraTotal      float64
	MonthsSaved     int
	InterestSaved   float64
}

// ================== Page Handlers ==================

func financeIndexHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "finance-index.html", nil)
	if err != nil {
		log.Println("Error executing finance index template:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func financeInvestmentHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "investment.html", nil)
	if err != nil {
		log.Println("Error executing investment template:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func financeReturnHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "return.html", nil)
	if err != nil {
		log.Println("Error executing return template:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func financeLoanHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "loan.html", nil)
	if err != nil {
		log.Println("Error executing loan template:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func financeSavingsGoalHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "savings-goal.html", nil)
	if err != nil {
		log.Println("Error executing savings-goal template:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func financeFIREHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "fire.html", nil)
	if err != nil {
		log.Println("Error executing fire template:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func financeInflationHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "inflation.html", nil)
	if err != nil {
		log.Println("Error executing inflation template:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func financeDebtHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "debt.html", nil)
	if err != nil {
		log.Println("Error executing debt template:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// ================== Total Return Calculator ==================

func returnCalcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	initialInvestment, _ := strconv.ParseFloat(r.FormValue("initialInvestment"), 64)
	finalValue, _ := strconv.ParseFloat(r.FormValue("finalValue"), 64)
	years, _ := strconv.ParseFloat(r.FormValue("years"), 64)

	if initialInvestment <= 0 || finalValue <= 0 || years <= 0 {
		http.Error(w, "All values must be positive", http.StatusBadRequest)
		return
	}

	result := calculateReturn(ReturnRequest{
		InitialInvestment: initialInvestment,
		FinalValue:        finalValue,
		Years:             years,
	})

	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		err := templates.ExecuteTemplate(w, "return-result", result)
		if err != nil {
			log.Println("Error executing return template:", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func calculateReturn(req ReturnRequest) ReturnResult {
	totalReturn := req.FinalValue - req.InitialInvestment
	percentReturn := (totalReturn / req.InitialInvestment) * 100

	// CAGR formula: ((FV/PV)^(1/years) - 1) * 100
	annualizedReturn := (math.Pow(req.FinalValue/req.InitialInvestment, 1/req.Years) - 1) * 100

	return ReturnResult{
		TotalReturn:      totalReturn,
		PercentReturn:    percentReturn,
		AnnualizedReturn: annualizedReturn,
	}
}

// ================== Loan vs Investing Calculator ==================

func loanComparisonHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	loanAmount, _ := strconv.ParseFloat(r.FormValue("loanAmount"), 64)
	interestRate, _ := strconv.ParseFloat(r.FormValue("interestRate"), 64)
	termMonths, _ := strconv.Atoi(r.FormValue("termMonths"))
	investmentReturnRate, _ := strconv.ParseFloat(r.FormValue("investmentReturnRate"), 64)

	if loanAmount <= 0 || interestRate < 0 || termMonths <= 0 {
		http.Error(w, "Invalid input values", http.StatusBadRequest)
		return
	}

	result := calculateLoanComparison(LoanComparisonRequest{
		LoanAmount:           loanAmount,
		InterestRate:         interestRate,
		TermMonths:           termMonths,
		InvestmentReturnRate: investmentReturnRate,
	})

	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		err := templates.ExecuteTemplate(w, "loan-result", result)
		if err != nil {
			log.Println("Error executing loan template:", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func calculateLoanComparison(req LoanComparisonRequest) LoanComparisonResult {
	// Convert annual rates to monthly (divide by 100 for percentage, then by 12 for monthly)
	monthlyRate := (req.InterestRate / 100) / 12
	monthlyInvestRate := (req.InvestmentReturnRate / 100) / 12

	n := float64(req.TermMonths)

	// Calculate monthly payment using amortization formula:
	// M = P * [r(1+r)^n] / [(1+r)^n - 1]
	var monthlyPayment float64
	if monthlyRate == 0 {
		monthlyPayment = req.LoanAmount / n
	} else {
		monthlyPayment = req.LoanAmount * (monthlyRate * math.Pow(1+monthlyRate, n)) /
			(math.Pow(1+monthlyRate, n) - 1)
	}

	// Build amortization schedule
	schedule := make([]AmortizationRow, req.TermMonths)
	balance := req.LoanAmount
	investmentValue := 0.0
	totalPaid := 0.0
	totalInterest := 0.0

	for month := 1; month <= req.TermMonths; month++ {
		interest := balance * monthlyRate
		principal := monthlyPayment - interest
		balance -= principal

		// If investing the monthly payment instead
		investmentValue = (investmentValue + monthlyPayment) * (1 + monthlyInvestRate)

		totalPaid += monthlyPayment
		totalInterest += interest

		schedule[month-1] = AmortizationRow{
			Month:            month,
			Payment:          monthlyPayment,
			Principal:        principal,
			Interest:         interest,
			RemainingBalance: math.Max(0, balance),
			InvestmentValue:  investmentValue,
		}
	}

	return LoanComparisonResult{
		MonthlyPayment:   monthlyPayment,
		TotalPaid:        totalPaid,
		TotalInterest:    totalInterest,
		FinalInvestValue: investmentValue,
		Difference:       investmentValue - totalPaid,
		Schedule:         schedule,
	}
}

// ================== Investment Calculator with Yahoo Finance ==================

func investmentCalcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	ticker := r.FormValue("ticker")
	startDateStr := r.FormValue("startDate")
	endDateStr := r.FormValue("endDate")
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	frequency := r.FormValue("frequency")

	if ticker == "" || startDateStr == "" || endDateStr == "" || amount <= 0 {
		http.Error(w, "Invalid input values", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		http.Error(w, "Invalid start date format", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		http.Error(w, "Invalid end date format", http.StatusBadRequest)
		return
	}

	if frequency != "weekly" && frequency != "monthly" {
		frequency = "monthly"
	}

	// Fetch historical prices from Yahoo Finance
	prices, err := fetchHistoricalPrices(ticker, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching prices for %s: %v", ticker, err)
		if isHTMXRequest(r) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<p><strong>Error:</strong> Could not fetch data for ticker '%s'. Please check the symbol and try again.</p>", ticker)
			return
		}
		http.Error(w, "Error fetching stock data", http.StatusInternalServerError)
		return
	}

	if len(prices) == 0 {
		if isHTMXRequest(r) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<p><strong>Error:</strong> No price data found for '%s' in the specified date range.</p>", ticker)
			return
		}
		http.Error(w, "No price data found", http.StatusNotFound)
		return
	}

	result := calculateInvestmentGrowth(ticker, prices, amount, frequency)

	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		err := templates.ExecuteTemplate(w, "investment-result", result)
		if err != nil {
			log.Println("Error executing investment template:", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func fetchHistoricalPrices(ticker string, startDate, endDate time.Time) ([]StockPrice, error) {
	// Yahoo Finance API v8 endpoint (no API key required)
	url := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d",
		ticker,
		startDate.Unix(),
		endDate.Unix(),
	)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; PersonalWebsite/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Yahoo Finance returned status %d", resp.StatusCode)
	}

	// Parse Yahoo Finance JSON response
	var result struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					AdjClose []struct {
						AdjClose []float64 `json:"adjclose"`
					} `json:"adjclose"`
				} `json:"indicators"`
			} `json:"result"`
			Error *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Chart.Error != nil {
		return nil, fmt.Errorf("%s: %s", result.Chart.Error.Code, result.Chart.Error.Description)
	}

	// Convert to StockPrice slice
	prices := make([]StockPrice, 0)
	if len(result.Chart.Result) > 0 {
		r := result.Chart.Result[0]
		for i, ts := range r.Timestamp {
			if len(r.Indicators.AdjClose) > 0 && i < len(r.Indicators.AdjClose[0].AdjClose) {
				adjClose := r.Indicators.AdjClose[0].AdjClose[i]
				// Skip days with no data (NaN or 0)
				if adjClose > 0 && !math.IsNaN(adjClose) {
					prices = append(prices, StockPrice{
						Date:     time.Unix(ts, 0),
						AdjClose: adjClose,
					})
				}
			}
		}
	}

	return prices, nil
}

func calculateInvestmentGrowth(ticker string, prices []StockPrice, amount float64, frequency string) InvestmentResult {
	var totalShares float64
	var totalInvested float64
	var investments int

	// Determine investment interval in days
	var intervalDays int
	if frequency == "weekly" {
		intervalDays = 7
	} else {
		intervalDays = 30
	}

	lastInvestDate := prices[0].Date.AddDate(0, 0, -intervalDays)

	for _, price := range prices {
		// Check if it's time to invest
		daysSinceLast := price.Date.Sub(lastInvestDate).Hours() / 24
		if daysSinceLast >= float64(intervalDays) {
			sharesBought := amount / price.AdjClose
			totalShares += sharesBought
			totalInvested += amount
			lastInvestDate = price.Date
			investments++
		}
	}

	finalValue := totalShares * prices[len(prices)-1].AdjClose
	totalReturn := finalValue - totalInvested
	percentReturn := 0.0
	if totalInvested > 0 {
		percentReturn = (totalReturn / totalInvested) * 100
	}

	return InvestmentResult{
		Ticker:        ticker,
		StartDate:     prices[0].Date.Format("Jan 02, 2006"),
		EndDate:       prices[len(prices)-1].Date.Format("Jan 02, 2006"),
		FinalValue:    finalValue,
		TotalInvested: totalInvested,
		TotalReturn:   totalReturn,
		PercentReturn: percentReturn,
		TotalShares:   totalShares,
		Investments:   investments,
	}
}

// ================== Savings Goal Calculator ==================

func savingsGoalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	goalAmount, _ := strconv.ParseFloat(r.FormValue("goalAmount"), 64)
	currentAge, _ := strconv.Atoi(r.FormValue("currentAge"))
	targetAge, _ := strconv.Atoi(r.FormValue("targetAge"))
	currentSavings, _ := strconv.ParseFloat(r.FormValue("currentSavings"), 64)
	annualReturn, _ := strconv.ParseFloat(r.FormValue("annualReturn"), 64)

	if goalAmount <= 0 || targetAge <= currentAge {
		http.Error(w, "Invalid input values", http.StatusBadRequest)
		return
	}

	result := calculateSavingsGoal(SavingsGoalRequest{
		GoalAmount:     goalAmount,
		CurrentAge:     currentAge,
		TargetAge:      targetAge,
		CurrentSavings: currentSavings,
		AnnualReturn:   annualReturn,
	})

	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		err := templates.ExecuteTemplate(w, "savings-goal-result", result)
		if err != nil {
			log.Println("Error executing savings-goal template:", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func calculateSavingsGoal(req SavingsGoalRequest) SavingsGoalResult {
	years := req.TargetAge - req.CurrentAge
	monthlyRate := (req.AnnualReturn / 100) / 12
	totalMonths := years * 12

	// Future value of current savings
	currentSavingsGrowth := req.CurrentSavings * math.Pow(1+monthlyRate, float64(totalMonths))

	// Amount still needed from contributions
	amountNeeded := req.GoalAmount - currentSavingsGrowth

	// PMT formula: FV = PMT * [((1+r)^n - 1) / r]
	// Solve for PMT: PMT = FV * r / ((1+r)^n - 1)
	var monthlySavings float64
	if monthlyRate == 0 {
		monthlySavings = amountNeeded / float64(totalMonths)
	} else {
		monthlySavings = amountNeeded * monthlyRate / (math.Pow(1+monthlyRate, float64(totalMonths)) - 1)
	}

	if monthlySavings < 0 {
		monthlySavings = 0 // Current savings already exceed goal
	}

	totalContributions := monthlySavings * float64(totalMonths)
	interestEarned := req.GoalAmount - totalContributions - req.CurrentSavings

	return SavingsGoalResult{
		MonthlySavings:       monthlySavings,
		WeeklySavings:        monthlySavings * 12 / 52,
		YearlySavings:        monthlySavings * 12,
		YearsToGoal:          years,
		TotalContributions:   totalContributions,
		InterestEarned:       interestEarned,
		CurrentSavingsGrowth: currentSavingsGrowth,
	}
}

// ================== FIRE Calculator ==================

func fireCalcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	currentAge, _ := strconv.Atoi(r.FormValue("currentAge"))
	annualIncome, _ := strconv.ParseFloat(r.FormValue("annualIncome"), 64)
	annualExpenses, _ := strconv.ParseFloat(r.FormValue("annualExpenses"), 64)
	currentSavings, _ := strconv.ParseFloat(r.FormValue("currentSavings"), 64)
	annualReturn, _ := strconv.ParseFloat(r.FormValue("annualReturn"), 64)

	if annualIncome <= 0 || annualExpenses <= 0 || annualExpenses >= annualIncome {
		http.Error(w, "Invalid input: expenses must be less than income", http.StatusBadRequest)
		return
	}

	result := calculateFIRE(FIRERequest{
		CurrentAge:     currentAge,
		AnnualIncome:   annualIncome,
		AnnualExpenses: annualExpenses,
		CurrentSavings: currentSavings,
		AnnualReturn:   annualReturn,
	})

	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		err := templates.ExecuteTemplate(w, "fire-result", result)
		if err != nil {
			log.Println("Error executing fire template:", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func calculateFIRE(req FIRERequest) FIREResult {
	// FIRE number = 25x annual expenses (4% safe withdrawal rate)
	fireNumber := req.AnnualExpenses * 25

	annualSavings := req.AnnualIncome - req.AnnualExpenses
	savingsRate := (annualSavings / req.AnnualIncome) * 100
	monthlyRate := (req.AnnualReturn / 100) / 12
	monthlySavings := annualSavings / 12

	// Simulate month by month until we hit FIRE number
	portfolio := req.CurrentSavings
	months := 0
	maxMonths := 100 * 12 // Cap at 100 years

	for portfolio < fireNumber && months < maxMonths {
		portfolio = portfolio*(1+monthlyRate) + monthlySavings
		months++
	}

	yearsToFIRE := float64(months) / 12
	fireAge := req.CurrentAge + int(math.Ceil(yearsToFIRE))

	return FIREResult{
		FIRENumber:      fireNumber,
		YearsToFIRE:     yearsToFIRE,
		FIREAge:         fireAge,
		SavingsRate:     savingsRate,
		AnnualSavings:   annualSavings,
		MonthlySavings:  monthlySavings,
		PortfolioAtFIRE: portfolio,
	}
}

// ================== Inflation Calculator ==================

func inflationCalcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	years, _ := strconv.Atoi(r.FormValue("years"))
	inflationRate, _ := strconv.ParseFloat(r.FormValue("inflationRate"), 64)

	if amount <= 0 || years <= 0 {
		http.Error(w, "Invalid input values", http.StatusBadRequest)
		return
	}

	result := calculateInflation(InflationRequest{
		Amount:        amount,
		Years:         years,
		InflationRate: inflationRate,
	})

	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		err := templates.ExecuteTemplate(w, "inflation-result", result)
		if err != nil {
			log.Println("Error executing inflation template:", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func calculateInflation(req InflationRequest) InflationResult {
	// Future value needed = Current * (1 + inflation)^years
	futureValueNeeded := req.Amount * math.Pow(1+(req.InflationRate/100), float64(req.Years))
	purchasingPowerLost := futureValueNeeded - req.Amount
	percentLost := (purchasingPowerLost / req.Amount) * 100

	return InflationResult{
		CurrentAmount:       req.Amount,
		Years:               req.Years,
		FutureValueNeeded:   futureValueNeeded,
		PurchasingPowerLost: purchasingPowerLost,
		PercentLost:         percentLost,
	}
}

// ================== Debt Payoff Calculator ==================

func debtCalcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	debtAmount, _ := strconv.ParseFloat(r.FormValue("debtAmount"), 64)
	interestRate, _ := strconv.ParseFloat(r.FormValue("interestRate"), 64)
	minimumPayment, _ := strconv.ParseFloat(r.FormValue("minimumPayment"), 64)
	extraPayment, _ := strconv.ParseFloat(r.FormValue("extraPayment"), 64)

	if debtAmount <= 0 || minimumPayment <= 0 {
		http.Error(w, "Invalid input values", http.StatusBadRequest)
		return
	}

	result := calculateDebtPayoff(DebtRequest{
		DebtAmount:     debtAmount,
		InterestRate:   interestRate,
		MinimumPayment: minimumPayment,
		ExtraPayment:   extraPayment,
	})

	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		err := templates.ExecuteTemplate(w, "debt-result", result)
		if err != nil {
			log.Println("Error executing debt template:", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func calculateDebtPayoff(req DebtRequest) DebtResult {
	monthlyRate := (req.InterestRate / 100) / 12

	// Calculate with minimum payment only
	minMonths, minInterest := simulateDebtPayoff(req.DebtAmount, monthlyRate, req.MinimumPayment)
	minTotal := req.MinimumPayment * float64(minMonths)

	// Calculate with extra payment
	totalPayment := req.MinimumPayment + req.ExtraPayment
	extraMonths, extraInterest := simulateDebtPayoff(req.DebtAmount, monthlyRate, totalPayment)
	extraTotal := totalPayment * float64(extraMonths)

	return DebtResult{
		MinimumMonths:   minMonths,
		MinimumYears:    float64(minMonths) / 12,
		MinimumInterest: minInterest,
		MinimumTotal:    minTotal,
		ExtraPayment:    req.ExtraPayment,
		ExtraMonths:     extraMonths,
		ExtraYears:      float64(extraMonths) / 12,
		ExtraInterest:   extraInterest,
		ExtraTotal:      extraTotal,
		MonthsSaved:     minMonths - extraMonths,
		InterestSaved:   minInterest - extraInterest,
	}
}

func simulateDebtPayoff(balance, monthlyRate, payment float64) (months int, totalInterest float64) {
	maxMonths := 1000 // Cap to prevent infinite loops

	for balance > 0 && months < maxMonths {
		interest := balance * monthlyRate
		totalInterest += interest

		principal := payment - interest
		if principal <= 0 {
			// Payment doesn't cover interest - debt grows forever
			return maxMonths, totalInterest
		}

		balance -= principal
		months++

		if balance < 0 {
			balance = 0
		}
	}

	return months, totalInterest
}
