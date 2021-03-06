package space_test

import (
	"fmt"
	"testing"

	"context"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunRepoBBTest(t *testing.T) {
	suite.Run(t, &repoBBTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type repoBBTest struct {
	gormtestsupport.DBTestSuite
	repo space.Repository
}

func (s *repoBBTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.repo = space.NewRepository(s.DB)
}

func (s *repoBBTest) TestCreate() {
	s.T().Run("ok", func(t *testing.T) {
		// given an identity
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(1))
		// when creating space
		name := testsupport.CreateRandomValidTestName("test space")
		id := uuid.NewV4()
		newSpace := space.Space{
			ID:      id,
			Name:    name,
			OwnerId: fxt.Identities[0].ID,
		}
		sp, err := s.repo.Create(context.Background(), &newSpace)
		require.Nil(t, err)
		require.NotNil(t, sp)
		require.Equal(t, id, sp.ID)
		require.Equal(t, name, sp.Name)
		require.Equal(t, fxt.Identities[0].ID, sp.OwnerId)
	})
	s.T().Run("fail - empty space name", func(t *testing.T) {
		// given an identity
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(1))
		// when creating space
		newSpace := space.Space{
			Name:    "",
			OwnerId: fxt.Identities[0].ID,
		}
		sp, err := s.repo.Create(context.Background(), &newSpace)
		require.NotNil(t, err)
		require.IsType(t, errors.BadParameterError{}, err, "error was %v", err)
		require.Nil(t, sp)
	})
	s.T().Run("fail - same owner", func(t *testing.T) {
		// given a space
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		// when trying to create the same space again
		newSpace := *fxt.Spaces[0]
		newSpace.ID = uuid.NewV4()
		sp, err := s.repo.Create(s.Ctx, &newSpace)
		// then
		require.NotNil(t, err)
		require.Nil(t, sp)
		require.IsType(t, errors.DataConflictError{}, err, "error was %v", err)
	})
}

func (s *repoBBTest) TestLoad() {
	s.T().Run("existing space", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		sp, err := s.repo.Load(s.Ctx, fxt.Spaces[0].ID)
		require.Nil(t, err)
		require.NotNil(t, sp)
		require.True(t, (*fxt.Spaces[0]).Equal(*sp))
	})
	s.T().Run("non-existing space", func(t *testing.T) {
		sp, err := s.repo.Load(s.Ctx, uuid.NewV4())
		require.NotNil(t, err)
		require.Nil(t, sp)
	})
}

func (s *repoBBTest) TestCheckExists() {
	resource.Require(s.T(), resource.Database)
	s.T().Run("space exists", func(t *testing.T) {
		// given a space
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		// when checking for existence
		err := s.repo.CheckExists(s.Ctx, fxt.Spaces[0].ID.String())
		// then
		require.Nil(t, err)
	})
	s.T().Run("space doesn't exist", func(t *testing.T) {
		err := s.repo.CheckExists(context.Background(), uuid.NewV4().String())
		require.NotNil(t, err)
		require.IsType(t, errors.NotFoundError{}, err, "error was %v", err)
	})
}

func (s *repoBBTest) TestSave() {
	s.T().Run("ok", func(t *testing.T) {
		// given a space
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		// when updating the name
		newName := testsupport.CreateRandomValidTestName("new name")
		fxt.Spaces[0].Name = newName
		sp, err := s.repo.Save(s.Ctx, fxt.Spaces[0])
		require.Nil(t, err)
		require.NotNil(t, sp)
		require.Equal(t, newName, sp.Name)
	})
	s.T().Run("fail - empty name", func(t *testing.T) {
		// given a space
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		// when saving the space with an empty name
		fxt.Spaces[0].Name = ""
		sp, err := s.repo.Save(s.Ctx, fxt.Spaces[0])
		// then
		require.NotNil(t, err)
		require.IsType(t, errors.BadParameterError{}, err, "error was %v", err)
		require.Nil(t, sp)
	})
	s.T().Run("fail - name already used", func(t *testing.T) {
		// given two spaces
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(2))
		// when saving one of the spaces with the name of the other
		fxt.Spaces[0].Name = fxt.Spaces[1].Name
		sp, err := s.repo.Save(s.Ctx, fxt.Spaces[0])
		// then
		require.NotNil(t, err)
		require.IsType(t, errors.BadParameterError{}, err, "error was %v", err)
		require.Nil(t, sp)
	})
	s.T().Run("fail - space not existing", func(t *testing.T) {
		// given a space with a not existing ID
		p := space.Space{
			ID:      uuid.NewV4(),
			Version: 0,
			Name:    testsupport.CreateRandomValidTestName("some space"),
		}
		// when updating this space
		sp, err := s.repo.Save(s.Ctx, &p)
		// then
		require.NotNil(t, err)
		require.IsType(t, errors.NotFoundError{}, err, "error was %v", err)
		require.Nil(t, sp)
	})
}

func (s *repoBBTest) TestDelete() {
	s.T().Run("ok", func(t *testing.T) {
		// given a space
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		id := fxt.Spaces[0].ID
		// double check that we can load this space
		sp, err := s.repo.Load(s.Ctx, id)
		require.Nil(t, err)
		require.NotNil(t, sp)
		// when
		err = s.repo.Delete(s.Ctx, id)
		// then
		require.Nil(t, err)
		// double check that we can no longer load the space
		sp, err = s.repo.Load(s.Ctx, id)
		require.NotNil(t, err)
		require.IsType(t, errors.NotFoundError{}, err, "error was %v", err)
		require.Nil(t, sp)
	})
	s.T().Run("not found - not existing space ID", func(t *testing.T) {
		// given a not existing space ID
		nonExistingSpaceID := uuid.NewV4()
		// when
		err := s.repo.Delete(s.Ctx, nonExistingSpaceID)
		// then
		require.NotNil(t, err)
		require.IsType(t, errors.NotFoundError{}, err, "error was %v", err)
	})
	s.T().Run("not found - nil space ID", func(t *testing.T) {
		// given a not existing space ID
		nilSpaceID := uuid.Nil
		// when
		err := s.repo.Delete(s.Ctx, nilSpaceID)
		// then
		require.NotNil(t, err)
		require.IsType(t, errors.NotFoundError{}, err, "error was %v", err)
	})
}

func (s *repoBBTest) TestList() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		var start, length *int
		_, orgCount, _ := s.repo.List(s.Ctx, start, length)
		// create a space
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		// when listing
		updatedListOfSpaces, newCount, _ := s.repo.List(s.Ctx, start, length)
		// then make sure we can find the newly created space
		t.Log(fmt.Sprintf("Old count of spaces : %d , new count of spaces : %d", orgCount, newCount))
		foundNewSpaceInList := false
		for _, retrievedSpace := range updatedListOfSpaces {
			if retrievedSpace.ID == fxt.Spaces[0].ID {
				foundNewSpaceInList = true
			}
		}
		// then
		require.True(t, foundNewSpaceInList)
	})
	s.T().Run("do not return pointer to same object", func(t *testing.T) {
		// given two spaces
		_ = tf.NewTestFixture(t, s.DB, tf.Spaces(2))
		// when
		var start, length *int
		spaces, newCount, _ := s.repo.List(s.Ctx, start, length)
		// then
		require.True(t, newCount >= 2)
		require.NotEqual(t, spaces[0].Name, spaces[1].Name)
	})
}

func (s *repoBBTest) TestLoadByOwnerAndName() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		// when
		sp, err := s.repo.LoadByOwnerAndName(context.Background(), &fxt.Spaces[0].OwnerId, &fxt.Spaces[0].Name)
		// then
		require.Nil(t, err)
		require.NotNil(t, sp)
		require.True(t, (*fxt.Spaces[0]).Equal(*sp))
	})
	s.T().Run("not found - different owner", func(t *testing.T) {
		// given two identities and one space
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(2), tf.Spaces(1))
		// when loading an existing space by name but with a different owner
		sp, err := s.repo.LoadByOwnerAndName(context.Background(), &fxt.Identities[1].ID, &fxt.Spaces[0].Name)
		// then
		require.NotNil(t, err)
		require.IsType(t, errors.NotFoundError{}, err, "error was %v", err)
		require.Nil(t, sp)
	})
	s.T().Run("not found - non existing space name", func(t *testing.T) {
		// given two identities and one space
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(2), tf.Spaces(1))
		// when loading an existing space by name but with a different owner
		nonExistingSpaceName := testsupport.CreateRandomValidTestName("non existing space name")
		sp, err := s.repo.LoadByOwnerAndName(context.Background(), &fxt.Spaces[0].OwnerId, &nonExistingSpaceName)
		// then
		require.NotNil(t, err)
		require.IsType(t, errors.NotFoundError{}, err, "error was %v", err)
		require.Nil(t, sp)
	})
}
