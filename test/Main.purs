module Test.Main where

import Prelude

import Effect (Effect)
import Data.Maybe (Maybe(..))
import Effect.Console (log)
import Effect.Exception (error, message)
import Promise as Promise
import Promise.Internal as P
import Promise.Rejection as Rejection
import Test.Assert as Assert

foreign import delay :: Int -> Effect (P.Promise Int)
foreign import failAfter :: Int -> Effect (P.Promise Int)

main :: Effect Unit
main = do
  log "Testing Promise.new, then_, catch, finally, all, race"
  
  -- Test resolve
  _ <- Promise.new (\res _ -> res "success")
    >>= Promise.then_ (\msg -> do
        Assert.assertEqual { actual: msg, expected: "success" }
        pure (Promise.resolve unit)
      )
      
  -- Test reject and catch
  _ <- Promise.new (\(res :: Unit -> Effect Unit) rej -> rej (Rejection.fromError (error "fail")))
    >>= Promise.catch (\err -> do
        case Rejection.toError err of
          Just e -> Assert.assert' "Should be fail" (message e == "fail")
          Nothing -> Assert.assert' "Expected an Error" false
        pure (Promise.resolve unit)
      )
      
  -- Test all
  _ <- Promise.all [ Promise.resolve 1, Promise.resolve 2, Promise.resolve 3 ]
    >>= Promise.then_ (\arr -> do
        Assert.assertEqual { actual: arr, expected: [1, 2, 3] }
        pure (Promise.resolve unit)
      )
      
  -- Test race
  p1 <- failAfter 50
  p2 <- delay 10 >>= Promise.then_ (\_ -> pure (Promise.resolve 42))
  _ <- Promise.race [ p1, p2 ]
    >>= Promise.then_ (\res -> do
        Assert.assertEqual { actual: res, expected: 42 }
        pure (Promise.resolve unit)
      )

  log "Done!"
