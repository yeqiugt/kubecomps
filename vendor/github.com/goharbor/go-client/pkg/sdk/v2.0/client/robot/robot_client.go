// Code generated by go-swagger; DO NOT EDIT.

package robot

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

//go:generate mockery -name API -inpkg

// API is the interface of the robot client
type API interface {
	/*
	   CreateRobot creates a robot account

	   Create a robot account*/
	CreateRobot(ctx context.Context, params *CreateRobotParams) (*CreateRobotCreated, error)
	/*
	   DeleteRobot deletes a robot account

	   This endpoint deletes specific robot account information by robot ID.*/
	DeleteRobot(ctx context.Context, params *DeleteRobotParams) (*DeleteRobotOK, error)
	/*
	   GetRobotByID gets a robot account

	   This endpoint returns specific robot account information by robot ID.*/
	GetRobotByID(ctx context.Context, params *GetRobotByIDParams) (*GetRobotByIDOK, error)
	/*
	   ListRobot gets robot account

	   List the robot accounts with the specified level and project.*/
	ListRobot(ctx context.Context, params *ListRobotParams) (*ListRobotOK, error)
	/*
	   RefreshSec refreshes the robot secret

	   Refresh the robot secret*/
	RefreshSec(ctx context.Context, params *RefreshSecParams) (*RefreshSecOK, error)
	/*
	   UpdateRobot updates a robot account

	   This endpoint updates specific robot account information by robot ID.*/
	UpdateRobot(ctx context.Context, params *UpdateRobotParams) (*UpdateRobotOK, error)
}

// New creates a new robot API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry, authInfo runtime.ClientAuthInfoWriter) *Client {
	return &Client{
		transport: transport,
		formats:   formats,
		authInfo:  authInfo,
	}
}

/*
Client for robot API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
	authInfo  runtime.ClientAuthInfoWriter
}

/*
CreateRobot creates a robot account

Create a robot account
*/
func (a *Client) CreateRobot(ctx context.Context, params *CreateRobotParams) (*CreateRobotCreated, error) {

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "CreateRobot",
		Method:             "POST",
		PathPattern:        "/robots",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &CreateRobotReader{formats: a.formats},
		AuthInfo:           a.authInfo,
		Context:            ctx,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*CreateRobotCreated), nil

}

/*
DeleteRobot deletes a robot account

This endpoint deletes specific robot account information by robot ID.
*/
func (a *Client) DeleteRobot(ctx context.Context, params *DeleteRobotParams) (*DeleteRobotOK, error) {

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "DeleteRobot",
		Method:             "DELETE",
		PathPattern:        "/robots/{robot_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &DeleteRobotReader{formats: a.formats},
		AuthInfo:           a.authInfo,
		Context:            ctx,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*DeleteRobotOK), nil

}

/*
GetRobotByID gets a robot account

This endpoint returns specific robot account information by robot ID.
*/
func (a *Client) GetRobotByID(ctx context.Context, params *GetRobotByIDParams) (*GetRobotByIDOK, error) {

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "GetRobotByID",
		Method:             "GET",
		PathPattern:        "/robots/{robot_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &GetRobotByIDReader{formats: a.formats},
		AuthInfo:           a.authInfo,
		Context:            ctx,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*GetRobotByIDOK), nil

}

/*
ListRobot gets robot account

List the robot accounts with the specified level and project.
*/
func (a *Client) ListRobot(ctx context.Context, params *ListRobotParams) (*ListRobotOK, error) {

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ListRobot",
		Method:             "GET",
		PathPattern:        "/robots",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListRobotReader{formats: a.formats},
		AuthInfo:           a.authInfo,
		Context:            ctx,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListRobotOK), nil

}

/*
RefreshSec refreshes the robot secret

Refresh the robot secret
*/
func (a *Client) RefreshSec(ctx context.Context, params *RefreshSecParams) (*RefreshSecOK, error) {

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "RefreshSec",
		Method:             "PATCH",
		PathPattern:        "/robots/{robot_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &RefreshSecReader{formats: a.formats},
		AuthInfo:           a.authInfo,
		Context:            ctx,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*RefreshSecOK), nil

}

/*
UpdateRobot updates a robot account

This endpoint updates specific robot account information by robot ID.
*/
func (a *Client) UpdateRobot(ctx context.Context, params *UpdateRobotParams) (*UpdateRobotOK, error) {

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "UpdateRobot",
		Method:             "PUT",
		PathPattern:        "/robots/{robot_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &UpdateRobotReader{formats: a.formats},
		AuthInfo:           a.authInfo,
		Context:            ctx,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*UpdateRobotOK), nil

}