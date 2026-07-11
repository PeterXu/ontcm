package models

import (
	knowledgemodels "ontcm/internal/knowledge/models"
)

// Web API response models

// FormulaListResponse represents the response for listing formulas
type FormulaListResponse struct {
	Total   int                `json:"total"`
	Formulas []FormulaSummary  `json:"formulas"`
}

// FormulaSummary represents a brief formula overview
type FormulaSummary struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Meridian string       `json:"meridian"`
	KeySymptoms []string  `json:"key_symptoms"`
}

// FormulaDetailResponse represents detailed formula information
type FormulaDetailResponse struct {
	ID            string                          `json:"id"`
	Name          string                          `json:"name"`
	Meridian      string                          `json:"meridian"`
	Composition   []knowledgemodels.HerbDose      `json:"composition"`
	KeySymptoms   []knowledgemodels.FormulaSymptom `json:"key_symptoms"`
	DrugSyndromes []knowledgemodels.DrugSyndrome   `json:"drug_syndromes"`
	Preparation   string                          `json:"preparation,omitempty"`
	OriginalText  string                          `json:"original_text,omitempty"`
	Contraindications []string                    `json:"contraindications,omitempty"`
}

// FormulaSearchResponse represents search results
type FormulaSearchResponse struct {
	Query    string             `json:"query"`
	Total    int                `json:"total"`
	Results  []FormulaMatch     `json:"results"`
}

// FormulaMatch represents a matched formula with score
type FormulaMatch struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Meridian    string   `json:"meridian"`
	MatchScore  float64  `json:"match_score"`
	MatchedSymptoms []string `json:"matched_symptoms"`
}

// HerbListResponse represents the response for listing herbs
type HerbListResponse struct {
	Total  int           `json:"total"`
	Herbs  []HerbSummary `json:"herbs"`
}

// HerbSummary represents a brief herb overview
type HerbSummary struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Tier         string   `json:"tier"`
	Nature       string   `json:"nature"`
	MainMeridians []string `json:"main_meridians"`
}

// HerbDetailResponse represents detailed herb information
type HerbDetailResponse struct {
	ID            string                           `json:"id"`
	Name          string                           `json:"name"`
	Tier          string                           `json:"tier"`
	Properties    knowledgemodels.HerbProperties   `json:"properties"`
	MainMeridians []string                         `json:"main_meridians"`
	DrugSyndromes []knowledgemodels.HerbDrugSyndrome `json:"drug_syndromes"`
	CommonPairings []string                        `json:"common_pairings,omitempty"`
	Contraindications []string                     `json:"contraindications,omitempty"`
	Safety        *knowledgemodels.SafetyInfo      `json:"safety,omitempty"`
}

// HerbSearchResponse represents search results
type HerbSearchResponse struct {
	Query   string       `json:"query"`
	Total   int          `json:"total"`
	Results []HerbMatch  `json:"results"`
}

// HerbMatch represents a matched herb with details
type HerbMatch struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Tier     string   `json:"tier"`
	Matched  []string `json:"matched_fields"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	KnowledgeBase KnowledgeStatus `json:"knowledge_base"`
}

// KnowledgeStatus represents knowledge base status
type KnowledgeStatus struct {
	FormulasLoaded int `json:"formulas_loaded"`
	HerbsLoaded    int `json:"herbs_loaded"`
	IndexReady     bool `json:"index_ready"`
}