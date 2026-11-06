package board

import "github.com/dvpico/Programming-Learning-AR/Server/game"

type control interface{ control() }

type unstartedControl struct{}
type readyControl struct{ value game.Value }
type pendingControl struct{ action game.Action }
type finishedControl struct{}

func (unstartedControl) control() {}
func (readyControl) control()     {}
func (pendingControl) control()   {}
func (finishedControl) control()  {}

type phase interface{ phase() }

type unstartedBoard struct{}
type collectingBoard struct{}
type awaitingReplyBoard struct{ robotID game.RobotID }
type awaitingCommitBoard struct{}
type finishedBoard struct{}

func (unstartedBoard) phase()      {}
func (collectingBoard) phase()     {}
func (awaitingReplyBoard) phase()  {}
func (awaitingCommitBoard) phase() {}
func (finishedBoard) phase()       {}
