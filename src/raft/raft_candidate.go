package raft

import (
	"math"
	"strconv"
)

func (rf *Raft) initCandidate() {
	rf.callElection()
}

func (rf *Raft) callElection() {
	rf.mu.Lock()
	// candidate check
	if rf.phase != "candidate" {
		rf.mu.Unlock()
		P(rf.me, "x candidate | check 1")
		return
	}
	// set up vote
	rf.currentTerm++
	rf.votedFor = -1
	term := rf.currentTerm
	rf.mu.Unlock()
	votes := make(chan bool, len(rf.peers))
	voteCount := 0
	majority := int(math.Ceil(float64(len(rf.peers)) / 2))
	// request votes via RequestVote RPC
	replies := make([]RequestVoteReply, len(rf.peers))
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID == rf.me && (rf.votedFor == -1 || rf.votedFor == rf.me) {
			votes <- true
			rf.votedFor = rf.me
		} else {
			args := RequestVoteArgs{Term: term, CandidateID: rf.me}
			go rf.sendRequestVote(ID, &args, &replies[ID], votes)
		}
	}
	P(rf.me, "requested votes |", term)
	// await votes
	for i := 0; i < len(rf.peers); i++ {
		select {
		case vote := <-votes:
			if vote {
				voteCount++
			}
		case <-electionReset:
			rf.phaseChange("follower", false, "vote interrupt")
			P(rf.me, "x candidate")
			return
		}
		// case 3: explicit election timeout?
		if voteCount >= majority {
			break
		}
	}
	P(rf.me, "received votes")
	rf.mu.Lock()

	// candidate check 2
	if rf.phase != "candidate" {
		rf.mu.Unlock()
		P(rf.me, "x candidate | check 2")
		return
	}
	rf.mu.Unlock()

	// continue
	if voteCount >= majority {
		rf.phaseChange("leader", true, "elected ("+strconv.Itoa(voteCount)+" votes; term "+strconv.Itoa(rf.currentTerm)+")")
	} else {
		rf.phaseChange("follower", false, "lost vote")
	}
	P(rf.me, "x candidate")
}